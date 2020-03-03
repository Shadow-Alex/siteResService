package models

import (
	"github.com/astaxie/beego/orm"
)

type WcCargoMaterials struct {
	Id         int    `orm:"column(id);auto" description:"自增id"`
	CargoId    uint   `orm:"column(cargo_id)" description:"货品ID"`
	CargoExtId uint   `orm:"column(cargo_ext_id)" description:"货品投放区域ID"`
	Type       int8   `orm:"column(type)" description:"类型(目前是落地页),0:落地页"`
	ReptileId  uint   `orm:"column(reptile_id)" description:"抓取类ID(扩展类使用)"`
	PageLink   string `orm:"column(page_link);size(1000)" description:"抓取链接"`
	Cover      string `orm:"column(cover);size(500)" description:"封面图片"`
	Images     string `orm:"column(images);null" description:"轮播图片集"`
	Content    string `orm:"column(content);null" description:"物料描述(图文或者纯图片)"`
	Extend     string `orm:"column(extend);null" description:"扩展字段(存取获取的原始数据)"`
	CreateTime int    `orm:"column(create_time)" description:"创建时间"`
	UpdateTime int    `orm:"column(update_time)" description:"更新时间"`
}

// TableName return this model of table name
func (t *WcCargoMaterials) TableName() string {
	return "wc_cargo_materials"
}

// inti for package model
func init() {
	// register model
	orm.RegisterModel(new(WcCargoMaterials))
}