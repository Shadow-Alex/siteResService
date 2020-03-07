package models

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/astaxie/beego/orm"
)

type FbAds struct {
	Id                  int       `orm:"column(id);auto"`
	PageId              string    `orm:"column(page_id);size(128);null"`
	PageName            string    `orm:"column(page_name);size(128);null"`
	AdId                string    `orm:"column(ad_id);size(128)"`
	PostId              string    `orm:"column(post_id);size(128);null"`
	Title               string    `orm:"column(title);size(1024);null"`
	LinkUrl             string    `orm:"column(link_url);size(1024);null"`
	LinkUrlFmBody       string    `orm:"column(link_url_fm_body);size(255);null"`
	LinkTitle           string    `orm:"column(link_title);size(255);null" description:"落地页title"`
	LinkImageUrl        string    `orm:"column(link_image_url);size(512);null" description:"落地页image url"`
	PageUrlId           string    `orm:"column(page_url_id);size(64);null"`
	VideoImageUrl       string    `orm:"column(video_image_url);size(512);null"`
	VideoSdUrl          string    `orm:"column(video_sd_url);size(512);null"`
	PageLikeCount       int64     `orm:"column(page_like_count);null"`
	CreateAt            int64     `orm:"column(create_at);null"`
	StartAt             int64     `orm:"column(start_at);null"`
	Active              int8      `orm:"column(active);null"`
	TbRefId             int64     `orm:"column(tb_ref_id);null"`
	Status              int       `orm:"column(status);null"`
	AdSource            int       `orm:"column(ad_source);null" description:"0:fb广告资料库 1：adspy站"`
	CreateTime          time.Time `orm:"column(create_time);type(datetime);null;auto_now_add"`
	UpdateTime          time.Time `orm:"column(update_time);type(datetime);null;auto_now_add"`
	Keyword             string    `orm:"column(keyword);size(255);null"`
	Clicks              int64     `orm:"column(clicks);null"`
	Clicks7days         int64     `orm:"column(clicks_7days);null"`
	PostLikes           int       `orm:"column(post_likes);null" description:"帖子点赞"`
	PostComments        int       `orm:"column(post_comments);null" description:"帖子评论数"`
	PostShares          int       `orm:"column(post_shares);null" description:"帖子分享数"`
	ImageUrl            string    `orm:"column(image_url);null"`
	Body                string    `orm:"column(body);null"`
	PullTime            uint      `orm:"column(pull_time);null" description:"抓取时间"`
	IsShield            int8      `orm:"column(is_shield)" description:"是否屏蔽:0,不屏蔽;1,已屏蔽"`
	AdType              string    `orm:"column(ad_type);size(20);null" description:"系统标记状态"`
	PostNum             int       `orm:"column(post_num);null"`
	AdTypeReason        string    `orm:"column(ad_type_reason);size(255);null" description:"系统命中原因"`
	AdTypeForLabour     string    `orm:"column(ad_type_for_labour);size(20);null" description:"人工标记状态"`
	AdTypeForLabourUser int       `orm:"column(ad_type_for_labour_user);null" description:"人工标记人员"`
	AdTypeForLabourTime int       `orm:"column(ad_type_for_labour_time);null" description:"人工标记时间"`
	AdTypeForSysTime    int       `orm:"column(ad_type_for_sys_time);null" description:"系统标记时间"`
	IsError             int       `orm:"column(is_error);null" description:"落地页是否http通过"`
	Language            string    `orm:"column(language);size(10)" description:"语言"`
	LinkCurrency        string    `orm:"column(link_currency);size(512);null" description:"落地页货币"`
	LinkCate            int       `orm:"column(link_cate);null" description:"落地页品类(0-未分类，1-家居)"`
	MinCreateTime       int64     `orm:"column(min_create_time);null" description:"最早创建时间"`
	FanPageLevel        uint      `orm:"column(fan_page_level)" description:"粉丝页等级"`
	DupCheck            int       `orm:"column(dup_check);null" description:"重复检测（0-未检测，1-检测未重复，2-重复产品，11-未重复，图片上传 12-未重复，上传失败）"`
	AdIds               string    `orm:"column(ad_ids);null" description:"广告ID"`
	UpdateHistory       string    `orm:"column(update_history);null" description:"更新历史"`
	UpdateBy            string    `orm:"column(update_by);size(255);null" description:"最近一次由谁更新"`
}

func (t *FbAds) TableName() string {
	return "fb_ads"
}

func init() {
	orm.RegisterModel(new(FbAds))
}

// AddFbAds insert a new FbAds into database and returns
// last inserted Id on success.
func AddFbAds(m *FbAds) (id int64, err error) {
	o := orm.NewOrm()
	id, err = o.Insert(m)
	return
}

// GetFbAdsById retrieves FbAds by Id. Returns error if
// Id doesn't exist
func GetFbAdsById(id int) (v *FbAds, err error) {
	o := orm.NewOrm()
	//TODO
	o.Using("kyreport")
	v = &FbAds{Id: id}
	if err = o.Read(v); err == nil {
		return v, nil
	}
	return nil, err
}

// GetAllFbAds retrieves all FbAds matches certain condition. Returns empty list if
// no records exist
func GetAllFbAds(query map[string]string, fields []string, sortby []string, order []string,
	offset int64, limit int64) (ml []interface{}, err error) {
	o := orm.NewOrm()
	qs := o.QueryTable(new(FbAds))
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

	var l []FbAds
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

// UpdateFbAds updates FbAds by Id and returns error if
// the record to be updated doesn't exist
func UpdateFbAdsById(m *FbAds) (err error) {
	o := orm.NewOrm()
	v := FbAds{Id: m.Id}
	// ascertain id exists in the database
	if err = o.Read(&v); err == nil {
		var num int64
		if num, err = o.Update(m); err == nil {
			fmt.Println("Number of records updated in database:", num)
		}
	}
	return
}

// DeleteFbAds deletes FbAds by Id and returns error if
// the record to be deleted doesn't exist
func DeleteFbAds(id int) (err error) {
	o := orm.NewOrm()
	v := FbAds{Id: id}
	// ascertain id exists in the database
	if err = o.Read(&v); err == nil {
		var num int64
		if num, err = o.Delete(&FbAds{Id: id}); err == nil {
			fmt.Println("Number of records deleted in database:", num)
		}
	}
	return
}
