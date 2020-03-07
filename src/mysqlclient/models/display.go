package models

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/astaxie/beego/orm"
)

type Display struct {
	Id          int    `orm:"column(id);auto"`
	Title       string `orm:"column(title);size(512);null" description:"广告标题"`
	Desc        string `orm:"column(desc);size(512);null" description:"广告描述"`
	Cover       string `orm:"column(cover);null" description:"头图/轮播图/视频"`
	Price       int    `orm:"column(price);null" description:"现价"`
	OriginPrice int    `orm:"column(origin_price);null" description:"原价"`
	Currency    string `orm:"column(currency);size(8);null" description:"货币"`
	Detail      string `orm:"column(detail);null" description:"广告详情"`
	UpdateTime  uint64 `orm:"column(update_time);null"`
	CreateTime  uint64 `orm:"column(create_time);null"`
}

func (t *Display) TableName() string {
	return "display"
}

func init() {
	orm.RegisterModel(new(Display))
}

// AddDisplay insert a new Display into database and returns
// last inserted Id on success.
func AddDisplay(m *Display) (id int64, err error) {
	o := orm.NewOrm()
	id, err = o.Insert(m)
	return
}

// GetDisplayById retrieves Display by Id. Returns error if
// Id doesn't exist
func GetDisplayById(id int) (v *Display, err error) {
	o := orm.NewOrm()
	v = &Display{Id: id}
	if err = o.Read(v); err == nil {
		return v, nil
	}
	return nil, err
}

// GetAllDisplay retrieves all Display matches certain condition. Returns empty list if
// no records exist
func GetAllDisplay(query map[string]string, fields []string, sortby []string, order []string,
	offset int64, limit int64) (ml []interface{}, err error) {
	o := orm.NewOrm()
	qs := o.QueryTable(new(Display))
	// query k=v
	for k, v := range query {
		// rewrite dot-notation to Object__Attribute
		k = strings.Replace(k, ".", "__", -1)
		if strings.Contains(k, "isnull") {
			qs = qs.Filter(k, (v == "true" || v == "1"))
		} else {
			qs = qs.Filter(k, v)
		}
	}
	// order by:
	var sortFields []string
	if len(sortby) != 0 {
		if len(sortby) == len(order) {
			// 1) for each sort field, there is an associated order
			for i, v := range sortby {
				orderby := ""
				if order[i] == "desc" {
					orderby = "-" + v
				} else if order[i] == "asc" {
					orderby = v
				} else {
					return nil, errors.New("Error: Invalid order. Must be either [asc|desc]")
				}
				sortFields = append(sortFields, orderby)
			}
			qs = qs.OrderBy(sortFields...)
		} else if len(sortby) != len(order) && len(order) == 1 {
			// 2) there is exactly one order, all the sorted fields will be sorted by this order
			for _, v := range sortby {
				orderby := ""
				if order[0] == "desc" {
					orderby = "-" + v
				} else if order[0] == "asc" {
					orderby = v
				} else {
					return nil, errors.New("Error: Invalid order. Must be either [asc|desc]")
				}
				sortFields = append(sortFields, orderby)
			}
		} else if len(sortby) != len(order) && len(order) != 1 {
			return nil, errors.New("Error: 'sortby', 'order' sizes mismatch or 'order' size is not 1")
		}
	} else {
		if len(order) != 0 {
			return nil, errors.New("Error: unused 'order' fields")
		}
	}

	var l []Display
	qs = qs.OrderBy(sortFields...)
	if _, err = qs.Limit(limit, offset).All(&l, fields...); err == nil {
		if len(fields) == 0 {
			for _, v := range l {
				ml = append(ml, v)
			}
		} else {
			// trim unused fields
			for _, v := range l {
				m := make(map[string]interface{})
				val := reflect.ValueOf(v)
				for _, fname := range fields {
					m[fname] = val.FieldByName(fname).Interface()
				}
				ml = append(ml, m)
			}
		}
		return ml, nil
	}
	return nil, err
}

// UpdateDisplay updates Display by Id and returns error if
// the record to be updated doesn't exist
func UpdateDisplayById(m *Display) (err error) {
	o := orm.NewOrm()
	v := Display{Id: m.Id}
	// ascertain id exists in the database
	if err = o.Read(&v); err == nil {
		var num int64
		if num, err = o.Update(m); err == nil {
			fmt.Println("Number of records updated in database:", num)
		}
	}
	return
}

// DeleteDisplay deletes Display by Id and returns error if
// the record to be deleted doesn't exist
func DeleteDisplay(id int) (err error) {
	o := orm.NewOrm()
	v := Display{Id: id}
	// ascertain id exists in the database
	if err = o.Read(&v); err == nil {
		var num int64
		if num, err = o.Delete(&Display{Id: id}); err == nil {
			fmt.Println("Number of records deleted in database:", num)
		}
	}
	return
}
