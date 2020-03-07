/*
  Package db for database customized operations
*/

package db

import (
	"runtime/debug"
	"sync/atomic"

	"github.com/astaxie/beego/orm"
	log "github.com/sirupsen/logrus"
)

// CustomizedDBQueryColumn return num of query result and last query item's createTime
func (db *MySQLClient) CustomizedDBQueryColumn(table string, cond *orm.Condition, col string, offset int64) *[]orm.Params {
	defer func() { // add recover to catch panic
		if err := recover(); err != nil {
			log.WithFields(log.Fields{
				"table": 	  table,
				"offset":     offset,
				"limit":      db.limit,
				"error info": err,
			}).Fatal("query column failed") // err is panic incoming content of panic
			log.Fatal(string(debug.Stack()))
		}
	}()

	items := new([]orm.Params)
	db.QueryColumn(items, table, cond, "-create_time", col, offset)

	// only for debug
	log.WithFields(log.Fields{
		"offset":   offset,
		"num":  	len(*items),
	}).Debug("customized query column status")

	return items
}

func (db *MySQLClient) CustomizedDBQueryMultiColumn(table string, cond *orm.Condition, offset int64) *[]orm.Params {
	defer func() { // add recover to catch panic
		if err := recover(); err != nil {
			log.WithFields(log.Fields{
				"table": 	  table,
				"offset":     offset,
				"limit":      db.limit,
				"error info": err,
			}).Fatal("query column failed") // err is panic incoming content of panic
			log.Fatal(string(debug.Stack()))
		}
	}()

	// get query seter instance
	qs := db.querySeter(table, cond, offset)
	if qs == nil {
		return &[]orm.Params{}
	}

	// only query with conditions
	items := new([]orm.Params)
	num, errQ := (*qs).OrderBy("-create_time").Values(items, "id", "cargo_id", "landing_url", "create_time")
	if errQ != nil {
		log.WithFields(log.Fields{
			"alias":      	db.alias,
			"table":		table,
			"orderBy":		"-create_time",
			"cond":     	*cond,
			"offset":     	offset,
			"limit":     	db.limit,
			"error info": 	errQ.Error(),
		}).Error("query items of all column from mysql failed")

		return &[]orm.Params{}
	}

	atomic.AddUint64(&db.QueryCounter, 1) // count query db num

	//fieldsMap := new([]orm.Params)
	//for i := 0; i < len(*items); i++ {
	//	for _, col := range cols{
	//		(*fieldsMap)[i][col] = (*items)[i][col]
	//	}
	//}

	// only for debug
	log.WithFields(log.Fields{
		"offset":   offset,
		"num":  	num,
	}).Debug("customized query column status")

	return items
}

// CustomizedDBQueryMax return max of query field
func (db *MySQLClient) CustomizedDBQueryMax(table string, cond *orm.Condition, col string, offset int64) uint64 {
	defer func() { // add recover to catch panic
		if err := recover(); err != nil {
			log.WithFields(log.Fields{
				"table": 	  table,
				"offset":     offset,
				"limit":      db.limit,
				"error info": err,
			}).Fatal("query column failed") // err is panic incoming content of panic
			log.Fatal(string(debug.Stack()))
		}
	}()

	items := new([]orm.Params)
	orderExprs := "-" + col
	db.QueryColumn(items, table, cond, orderExprs, col, offset)

	return (*items)[0]["CargoExtId"].(uint64)
}