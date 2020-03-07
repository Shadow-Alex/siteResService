package models

import (
	"github.com/astaxie/beego/orm"
)

type WcCargoReptileTask struct {
	Id         int    `orm:"column(id);auto" description:"自增id"`
	CargoId    uint   `orm:"column(cargo_id)" description:"货品ID"`
	CargoExtId uint   `orm:"column(cargo_ext_id)" description:"货品投放区域ID"`
	Type       int8   `orm:"column(type)" description:"类型(目前是落地页),0:落地页"`
	ReptileId  uint   `orm:"column(reptile_id)" description:"抓取类ID(扩展类使用)"`
	PageLink   string `orm:"column(page_link);size(1000)" description:"抓取链接"`
	Status     int8   `orm:"column(status)" description:"状态:0,未处理;1,处理中;2,已处理;3,已取消;4,失败重试"`
	AdminId    uint   `orm:"column(admin_id);null" description:"操作人ID"`
	Extend     string `orm:"column(extend);null" description:"扩展字段"`
	CreateTime int    `orm:"column(create_time)" description:"创建时间"`
	UpdateTime int    `orm:"column(update_time)" description:"更新时间"`
}

// TableName return this model of table name
func (t *WcCargoReptileTask) TableName() string {
	return "wc_cargo_reptile_task"
}

// inti for package model
func init() {
	// register model
	orm.RegisterModel(new(WcCargoReptileTask))
}