package models

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/astaxie/beego/orm"
)

type Admin struct {
	Id         int    `orm:"column(id);auto"`
	UserName   string `orm:"column(user_name);size(255);null"`
	Email      string `orm:"column(email);size(255);null"`
	RoleId     uint   `orm:"column(role_id);null" description:"角色ID"`
	Password   string `orm:"column(password);size(32);null"`
	Salt       string `orm:"column(salt);size(10);null"`
	Truename   string `orm:"column(truename);size(255);null"`
	ParentId   int    `orm:"column(parent_id);null"`
	Phone      string `orm:"column(phone);size(255);null"`
	Type       string `orm:"column(type);size(2)" description:"投手类型：人工 0、机器复制 1、机器新品 2"`
	CreateTime uint64 `orm:"column(create_time);null"`
	UpdateTime uint64 `orm:"column(update_time);null"`
}

func (t *Admin) TableName() string {
	return "admin"
}

func init() {
	orm.RegisterModel(new(Admin))
}

// AddAdmin insert a new Admin into database and returns
// last inserted Id on success.
func AddAdmin(m *Admin) (id int64, err error) {
	o := orm.NewOrm()
	id, err = o.Insert(m)
	return
}

// GetAdminById retrieves Admin by Id. Returns error if
// Id doesn't exist
func GetAdminById(id int) (v *Admin, err error) {
	o := orm.NewOrm()
	v = &Admin{Id: id}
	if err = o.Read(v); err == nil {
		return v, nil
	}
	return nil, err
}

// GetAllAdmin retrieves all Admin matches certain condition. Returns empty list if
// no records exist
func GetAllAdmin(query map[string]string, fields []string, sortby []string, order []string,
	offset int64, limit int64) (ml []interface{}, err error) {
	o := orm.NewOrm()
	qs := o.QueryTable(new(Admin))
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

	var l []Admin
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

// UpdateAdmin updates Admin by Id and returns error if
// the record to be updated doesn't exist
func UpdateAdminById(m *Admin) (err error) {
	o := orm.NewOrm()
	v := Admin{Id: m.Id}
	// ascertain id exists in the database
	if err = o.Read(&v); err == nil {
		var num int64
		if num, err = o.Update(m); err == nil {
			fmt.Println("Number of records updated in database:", num)
		}
	}
	return
}

// DeleteAdmin deletes Admin by Id and returns error if
// the record to be deleted doesn't exist
func DeleteAdmin(id int) (err error) {
	o := orm.NewOrm()
	v := Admin{Id: id}
	// ascertain id exists in the database
	if err = o.Read(&v); err == nil {
		var num int64
		if num, err = o.Delete(&Admin{Id: id}); err == nil {
			fmt.Println("Number of records deleted in database:", num)
		}
	}
	return
}
