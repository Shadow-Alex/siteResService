package models

import (
	"github.com/astaxie/beego/orm"
)

type WcCargoExt struct {
	Id                  int     `orm:"column(id);auto" description:"自增id"`
	CargoId             uint    `orm:"column(cargo_id)" description:"外键、货品id"`
	MoneyUnitId         int     `orm:"column(money_unit_id)" description:"货币ID"`
	MoneyUnit           string  `orm:"column(money_unit);size(30)" description:"货币单位"`
	CostPrice           float64 `orm:"column(cost_price);digits(15);decimals(2)" description:"成本价格"`
	CostMoney           string  `orm:"column(cost_money);size(255)" description:"产品成本"`
	ComparePrice        float64 `orm:"column(compare_price);null;digits(8);decimals(2)" description:"市场价格(对比价格)"`
	Price               string  `orm:"column(price);size(255)" description:"产品价格（产品套餐售价）"`
	ProfitRate          string  `orm:"column(profit_rate);size(255)" description:"利润率"`
	Sort                int     `orm:"column(sort)" description:"排序(同品类或者限定条件时，可控制产品的显示顺序)"`
	VideoUrl            string  `orm:"column(video_url);size(800)" description:"视频参考链接"`
	LandingUrl          string  `orm:"column(landing_url);size(800)" description:"落地页链接"`
	SupplierUrl         string  `orm:"column(supplier_url);size(1000)" description:"供应商链接"`
	CompetitorUrl       string  `orm:"column(competitor_url);size(800)" description:"竞品链接"`
	PagedesignRequire   string  `orm:"column(pagedesign_require);size(1000);null" description:"页面设计要求"`
	PitcherNote         string  `orm:"column(pitcher_note);size(1000)" description:"投手备注"`
	Type                uint8   `orm:"column(type)" description:"创建来源：1cod；2shopify"`
	IsDesign            uint8   `orm:"column(is_design)" description:"是否需要设计：0否；1是"`
	ShipMoney           float64 `orm:"column(ship_money);digits(15);decimals(2)" description:"运费"`
	RejectRate          string  `orm:"column(reject_rate);size(255)" description:"拒付率"`
	Source              uint    `orm:"column(source);null" description:"0手动创建；1程序复制"`
	Weight              string  `orm:"column(weight);size(20);null" description:"货品重量"`
	AuditStatus         uint8   `orm:"column(audit_status)" description:"审核状态：0待审核；1审核中；2通过；3拒绝"`
	AuditTime           uint    `orm:"column(audit_time);null" description:"审核时间"`
	AuditUserId         uint    `orm:"column(audit_user_id);null" description:"审核人员id"`
	BuyRemark           string  `orm:"column(buy_remark);null" description:"购买备注"`
	UserId              uint    `orm:"column(user_id);null" description:"创建者id"`
	RefuseReason        string  `orm:"column(refuse_reason);null" description:"拒绝原因"`
	Tags                string  `orm:"column(tags);size(255)" description:"shopify 使用"`
	Collections         string  `orm:"column(collections);size(255)" description:"shopify 使用，携带入产品shopify 使用，携带入产品的collections"`
	ProductType         string  `orm:"column(product_type);size(255)" description:"shopify 使用，携带入产品的category_name"`
	Vendor              string  `orm:"column(vendor);size(255)" description:"shopify 使用，携带入产品的vendor"`
	CreateTime          int     `orm:"column(create_time)" description:"创建时间"`
	UpdateTime          int     `orm:"column(update_time)" description:"更新时间"`
	IsDel               int8    `orm:"column(is_del)" description:"是否删除0否1是，删除的产品在各个列表中不展示"`
	LandingPageIco      string  `orm:"column(landing_page_ico);size(255);null"`
	Channel             int8    `orm:"column(channel)" description:"渠道，0选品手动创建，1投手根据广告创建"`
	SelAdsId            int64   `orm:"column(sel_ads_id)" description:"备选广告自增ID"`
	SelAdsRecordId      int     `orm:"column(sel_ads_record_id)" description:"备选广告选取记录ID"`
	CargoComplete       int     `orm:"column(cargo_complete)" description:"货品完善状态：0未处理，1已创建产品，2已投放，3已完善(有货源货品，默认为已完善)"`
	CompleteAuditStatus uint    `orm:"column(complete_audit_status)" description:"完善信息审核审核状态：审核状态：0待审核；1审核中；2通过；3拒绝"`
	CompleteUid         int     `orm:"column(complete_uid)" description:"完善信息提审人员"`
	CompleteTime        int     `orm:"column(complete_time)" description:"完善信息提审时间"`
	StopLossCpo         string  `orm:"column(stop_loss_cpo);size(10)" description:"止损cpo"`
	AdDisposeId         int     `orm:"column(ad_dispose_id)" description:"ad_dispose表主键id"`
}

// TableName return this model of table name
func (t *WcCargoExt) TableName() string {
	return "wc_cargo_ext"
}

// inti for package model
func init() {
	// register model
	orm.RegisterModel(new(WcCargoExt))
}