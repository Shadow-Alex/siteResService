/*
  Package db for database operations
*/

package db

import (
	"sync"
	"sync/atomic"
	"time"
	"strings"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/orm"
	_ "github.com/go-sql-driver/mysql"
	log "github.com/sirupsen/logrus"

	cm "../common"
)

// MySQLClient represents mysql client
type MySQLClient struct {
	orm           orm.Ormer
	alias		  string  // database alias name
	lock          *sync.RWMutex
	itemList      []interface{}         // for multi insert
	limit 		  int  // for query show limit
	DataChan	  chan interface{}  // store query data
	InsertCounter uint64  // calculation num of insert mysql operation, must use by atomic !!!
	UpdateCounter uint64  // calculation num of update mysql operation, must use by atomic !!!
	QueryCounter  uint64  // calculation num of query mysql operation, must use by atomic !!!
	DeleteCounter  uint64  // calculation num of delete mysql operation, must use by atomic !!!
}

var retryCount int
var retryDelay int

// GetMySQLClientInstance returns MySQLClient instance pointer if create MySQL client success
func GetMySQLClientInstance(dbConns string, name ...string) *MySQLClient {
	alias := "default"
	if len(name) > 0 {
		alias = name[0]
	}
	instance := new(MySQLClient)
	instance.init(dbConns, alias)

	return instance
}

// init MySQL client
func (db *MySQLClient) init(dbConns string, alias string) {
	dbMaxIdleConns := beego.AppConfig.DefaultInt("mysql::connections.maxIdle", cm.DBMaxIdleCONNS)
	dbMaxOpenConns := beego.AppConfig.DefaultInt("mysql::connections.maxOpen", cm.DBMaxOpenCONNS)
	size := beego.AppConfig.DefaultInt("channelSize", cm.MaxChannelSize)

	db.limit = beego.AppConfig.DefaultInt("mysql::query.limit", cm.DBQueryLimit)
	retryCount = beego.AppConfig.DefaultInt("mysql::retry.count", cm.DBRetryCount)
	retryDelay = beego.AppConfig.DefaultInt("mysql::retry.delay", cm.DBRetryDelay)

	// register data driver, mysql / sqlite3 / postgres these three types have been registered by default, so it is unnecessary to set them
	orm.RegisterDriver("mysql", orm.DRMySQL)
	// register database, ORM must register a database with the alias "default" as the default
	err := orm.RegisterDataBase(alias, "mysql", dbConns)
	if err != nil {
		log.WithFields(log.Fields{
			"error info": err.Error(),
		}).Fatal("register database failed")

		return
	}
	// RunSyncdb auto create table, param 2 means whether open create table; param 3 means whether update table
	// note !!! : if param 2 is true, if table exist and has value,then it will delete the table first then create a new table, so param 2 may cause our data lost
	// orm.RunSyncdb("default", false, true)
	orm.SetMaxIdleConns(alias, dbMaxIdleConns)
	orm.SetMaxOpenConns(alias, dbMaxOpenConns)

	db.orm = orm.NewOrm()

	db.alias = alias  // keep alias name for switch database when operate db

	db.lock = new(sync.RWMutex)

	db.DataChan = make(chan interface{}, size)

	atomic.StoreUint64(&db.InsertCounter, 0) // init counter to 0
	atomic.StoreUint64(&db.UpdateCounter, 0) // init counter to 0
	atomic.StoreUint64(&db.QueryCounter, 0) // init counter to 0
	atomic.StoreUint64(&db.DeleteCounter, 0) // init counter to 0

	log.WithFields(log.Fields{
		"aliasName":      	db.alias,
		"queryLimit": 		db.limit,
		"dbMaxIdleConns": 	dbMaxIdleConns,
		"dbMaxOpenConns": 	dbMaxOpenConns,
		"retryCount": 		retryCount,
		"retryDelay": 		retryDelay,
	}).Info("init db client success...")
}

// isExist return true if item is exist in db
func (db *MySQLClient) isExist(table string, cond *orm.Condition) bool {
	db.lock.Lock()
	defer db.lock.Unlock()

	// must switch to specified database when operate multi db !!!
	errU := db.orm.Using(db.alias)
	if errU != nil {
		log.WithFields(log.Fields{
			"alias":      db.alias,
			"error": errU.Error(),
		}).Error("can not switch database")

		return false
	}

	return db.orm.QueryTable(table).SetCond(cond).Exist()
}

// IsExist return true if item is exist in db
func (db *MySQLClient) IsExist(table string, cond *orm.Condition) bool {
	return db.isExist(table, cond)
}

// insert item into mysql
func (db *MySQLClient) insert(item interface{}) error {
	db.lock.Lock()
	defer db.lock.Unlock()

	// must switch to specified database when operate multi db !!!
	errU := db.orm.Using(db.alias)
	if errU != nil {
		log.WithFields(log.Fields{
			"alias":      db.alias,
			"error": errU.Error(),
		}).Error("can not switch database")

		return errU
	}

	_, errI := db.orm.Insert(item)
	if errI != nil {
		log.WithFields(log.Fields{
			"alias":      db.alias,
			"error": errI.Error(),
		}).Error("insert one item into mysql failed")

		return errI
	}

	return nil
}

// SingleInsert for insert one item into mysql
func (db *MySQLClient) SingleInsert(item interface{}) bool {
	for i := 0; i < retryCount; i++ {  // if insert db failed, then retry
		err := db.insert(item)
		if err == nil {
			atomic.AddUint64(&db.InsertCounter, 1) // count insert db num

			return true
		} else if strings.Contains(err.Error(), "Duplicate entry") {  // if Duplicate entry, return false directly
			return false
		}

		time.Sleep(time.Duration(retryDelay) * time.Second)  // delay when retry
	}

	return false
}

// multiInsert for insert multi item into mysql
func (db *MySQLClient) multiInsert() error {
	db.lock.Lock()
	defer db.lock.Unlock()

	// must switch to specified database when operate multi db !!!
	errU := db.orm.Using(db.alias)
	if errU != nil {
		log.WithFields(log.Fields{
			"alias":      db.alias,
			"error": errU.Error(),
		}).Error("can not switch database")

		return errU
	}

	itemNum := len(db.itemList)
	_, errI := db.orm.InsertMulti(itemNum, db.itemList)
	if errI != nil {
		log.WithFields(log.Fields{
			"alias":      db.alias,
			"item num":    itemNum,
			"error":  errI.Error(),
		}).Error("insert multi item into mysql failed")

		return errI
	}

	return nil
}

// TODO: need to test more
// MultiInsert for insert multi item to mysql
func (db *MySQLClient) MultiInsert(item interface{}) bool {
	db.itemList = append(db.itemList, item)
	itemNum := len(db.itemList)
	if itemNum > cm.DBMultiInsertSize {
		for i := 0; i < retryCount; i++ { // if insert db failed, then retry
			err := db.multiInsert()
			if err == nil {
				// clear data item list after the itemList inserted into mysql
				db.itemList = db.itemList[0:0]

				atomic.AddUint64(&db.InsertCounter, uint64(itemNum)) // count insert db num

				return true
			} else if strings.Contains(err.Error(), "Duplicate entry") { // if Duplicate entry, return false directly
				return false
			}

			time.Sleep(time.Duration(retryDelay) * time.Second) // delay when retry
		}
	}

	return false
}

// update for update specified field
func (db *MySQLClient) update(table string, cond *orm.Condition, fields *orm.Params) error {
	db.lock.Lock()
	defer db.lock.Unlock()

	// must switch to specified database when operate multi db !!!
	errU := db.orm.Using(db.alias)
	if errU != nil {
		log.WithFields(log.Fields{
			"alias":      db.alias,
			"error": errU.Error(),
		}).Error("can not switch database")

		return errU
	}

	// only update specified field
	//_, errQ := db.orm.QueryTable(table).Filter(filterK, filterV).Update(fields)
	_, errQ := db.orm.QueryTable(table).SetCond(cond).Update(*fields)
	if errQ != nil {
		log.WithFields(log.Fields{
			"alias":      	db.alias,
			"condition":    *cond,
			"fields":     	*fields,
			"error": errQ.Error(),
		}).Error("update specified item from mysql failed")

		return errQ
	}

	return nil
}

// UpdateSpecifiedField for update specified field
func (db *MySQLClient) UpdateField(table string, cond *orm.Condition, fields *orm.Params) bool {
	for i := 0; i < retryCount; i++ {  // if insert db failed, then retry
		err := db.update(table, cond, fields)
		if err == nil {
			atomic.AddUint64(&db.UpdateCounter, 1) // count update db num

			return true
		}

		time.Sleep(time.Duration(retryDelay) * time.Second)  // delay when retry
	}

	return false
}

// querySeter for query specified field from offset and show limit
func (db *MySQLClient) querySeter(table string, cond *orm.Condition, offset int64) *orm.QuerySeter {
	db.lock.Lock()
	defer db.lock.Unlock()

	// must switch to specified database when operate multi db !!!
	errU := db.orm.Using(db.alias)
	if errU != nil {
		log.WithFields(log.Fields{
			"alias":      db.alias,
			"error": errU.Error(),
		}).Error("can not switch database")

		return nil
	}

	// only query with conditions
	qs := db.orm.QueryTable(table).SetCond(cond).Offset(offset).Limit(db.limit)
	if qs == nil {
		log.WithFields(log.Fields{
			"alias":      	db.alias,
			"table":		table,
			"cond":     	*cond,
			"offset":     	offset,
			"limit":     	db.limit,
		}).Error("get query seter from mysql failed")

		return nil
	}

	return &qs
}

// QueryAll for query all columns from offset and show limit
func (db *MySQLClient) QueryAll(items *[]orm.Params, table string, cond *orm.Condition, orderExprs string, offset int64) (int64, bool) {
	// get query seter instance
	qs := db.querySeter(table, cond, offset)
	if qs == nil {
		return 0, false
	}
	// only query with conditions
	num, errQ := (*qs).OrderBy(orderExprs).Values(items)
	if errQ != nil {
		log.WithFields(log.Fields{
			"alias":      	db.alias,
			"table":		table,
			"orderBy":		orderExprs,
			"cond":     	*cond,
			"offset":     	offset,
			"limit":     	db.limit,
			"error": 	errQ.Error(),
		}).Error("query items of all column from mysql failed")

		return 0, false
	}

	atomic.AddUint64(&db.QueryCounter, 1) // count query db num

	return num, true
}

// QueryColumns for query specified columns from offset and show limit
func (db *MySQLClient) QueryColumn(items *[]orm.Params, table string, cond *orm.Condition, orderExprs string, col string, offset int64) (int64, bool) {
	// get query seter instance
	qs := db.querySeter(table, cond, offset)
	if qs == nil {
		return 0, false
	}
	// only query with conditions
	num, errQ := (*qs).OrderBy(orderExprs).Values(items, col)
	if errQ != nil {
		log.WithFields(log.Fields{
			"alias":      	db.alias,
			"table":		table,
			"orderBy":		orderExprs,
			"column": 		col,
			"cond":     	*cond,
			"offset":     	offset,
			"limit":     	db.limit,
			"error": 	errQ.Error(),
		}).Error("query items of specified column from mysql failed")

		return 0, false
	}

	atomic.AddUint64(&db.QueryCounter, 1) // count query db num

	return num, true
}

func (db *MySQLClient) Delete(item interface{}, table string) (int64, bool) {
	db.lock.Lock()
	defer db.lock.Unlock()

	// must switch to specified database when operate multi db !!!
	errU := db.orm.Using(db.alias)
	if errU != nil {
		log.WithFields(log.Fields{
			"alias":      db.alias,
			"error": errU.Error(),
		}).Error("can not switch database")

		return 0, false
	}

	num, errD := db.orm.Delete(item)
	if errD != nil {
		log.WithFields(log.Fields{
			"alias":      	db.alias,
			"table":		table,
			"item":			item,
			"error": 	errD.Error(),
		}).Error("delete item from mysql failed")

		return 0, false
	}

	atomic.AddUint64(&db.DeleteCounter, 1) // count delete db num

	return num, true
}