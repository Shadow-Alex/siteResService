/*
  Package task for seek suggestion with specified page id using http service
*/

package taskservice

import (
	"net/url"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	jsoniter "github.com/json-iterator/go"
	log "github.com/sirupsen/logrus"
	"github.com/tebeka/selenium"

	cm "siteResService/src/common"
	md "siteResService/src/mysqlclient/models"
	ut "siteResService/src/util"
)

// redirectToOrderPage for click order and jump to order page, return yes if redirect success
func redirectOrderPage (wd *selenium.WebDriver, pageURL string, orderLabel string) bool {
	anchors, err := (*wd).FindElements(selenium.ByCSSSelector, orderLabel)
	if err != nil {
		log.WithFields(log.Fields{
			"url":		pageURL,
			"cssValue":	orderLabel,
			"error":	err.Error(),
		}).Error("can not find css element by redirectOrderPage")

		return false
	}

	// click href
	clickFlag := false
	for _, a := range anchors {
		err := a.Click()
		if err != nil {
			log.WithFields(log.Fields{
				"url":		pageURL,
				"error":	err.Error(),
			}).Error("send keys failed by redirectOrderPage")

			continue
		}

		clickFlag = true

		break
	}

	// if click failed, return
	if !clickFlag {
		log.Error("click order failed by redirectOrderPage")

		return false
	}

	return true
}

// getDocWebDriver returns pointer of goquery.Document instance by web driver
func getDocWebDriver(wd *selenium.WebDriver, pageURL string) *goquery.Document {
	source, errP := (*wd).PageSource()
	if errP != nil {
		log.WithFields(log.Fields{
			"pageURL":	pageURL,
			"error":	errP.Error(),
		}).Error("can not get page source by getDocWebDriver")

		return nil
	}

	// get page body html document
	html := source
	doc, errD := goquery.NewDocumentFromReader(strings.NewReader(html))
	if errD != nil {
		log.WithFields(log.Fields{
			"pageURL":	pageURL,
			"error":	errD.Error(),
		}).Error("new document failed by getDocWebDriver")

		return nil
	}

	return doc
}

// requestDocWebDriver returns currentURL and pointer of main page and order page body goquery.Document by web driver
func (t *TaskService) requestDocWebDriver(pageURL string) (string, *goquery.Document, *goquery.Document) {
	wd := t.httpService.GetURLWebDriver(pageURL)
	if wd == nil {
		log.WithFields(log.Fields{
			"pageURL":	pageURL,
		}).Error("get web driver failed by requestDocWebDriver")

		return "", nil, nil
	}

	currentURL, errU := (*wd).CurrentURL()
	if errU != nil {
		log.WithFields(log.Fields{
			"pageURL":	pageURL,
			"error":	errU.Error(),
		}).Error("can not get current url by requestDocWebDriver")

		return "", nil, nil
	}

	// get main page doc
	doc := getDocWebDriver(wd, pageURL)
	if doc == nil {
		return currentURL, nil, nil
	}

	// get order label
	u, _ := url.Parse(currentURL)
	domainMD5 := ut.GetMD5(u.Host)
	value, ok := t.site.SitesLabelMaps.Load(domainMD5)
	if !ok {
		log.WithFields(log.Fields{
			"url":		currentURL,
			"domain":	u.Host,
		}).Error("do not contains this domain template by requestDocWebDriver")

		return currentURL, doc, nil
	}

	// redirect order page
	labels := value.(*cm.LabelsParse)
	if len(labels.Order) <= 0 {
		log.Info("do not need order page by requestDocWebDriver")

		return currentURL, doc, nil
	}

	redirect := false  // for judge redirect status
	for _, labelOrder := range labels.Order {
		if len(labelOrder) <= 0 {
			continue
		}

		if redirectOrderPage(wd, pageURL, labelOrder) {
			redirect = true
			// waite to complete load the page
			time.Sleep(time.Duration(3) * time.Second)

			break
		}
	}

	// get order page doc only if redirect success
	if redirect {
		orderDoc := getDocWebDriver(wd, pageURL)
		if orderDoc != nil {
			return currentURL, doc, orderDoc
		}
	}

	log.Error("web driver load order page failed by requestDocWebDriver")

	return currentURL, doc, nil
}


// getOrderHref returns order href
func getOrderHref(doc *goquery.Document, pageURL string, orderLabel string) string {
	selectors := strings.Split(orderLabel, cm.LabelSeparate)
	selection := doc.Find(selectors[0])
	for i := 1; i < len(selectors); i++ {  // cascade find label
		selection = selection.Find(selectors[i])
	}

	// find order href
	var href string
	selection.Each(func(i int, selc *goquery.Selection) {
		href, _ = selection.Attr("href")
	})

	// get order href
	if len(href) > 0 {
		index := strings.LastIndex(pageURL, "/")
		if index > 0 {
			return pageURL[: index] + href
		}

		return href
	}

	return ""
}

// requestDocHTTP returns main doc and order doc pointer of goquery.Document instance by http request get
func (t *TaskService) requestDocHTTP(pageURL string, orderLabels []string) (*goquery.Document, *goquery.Document) {
	doc := t.httpService.GetDocRequestGet(pageURL)
	if doc == nil {
		return nil, nil
	}

	var orderURL string
	for _, label := range orderLabels {
		if len(label) <= 0 {
			continue
		}

		// get order doc
		orderURL = getOrderHref(doc, pageURL, label)
		if len(orderURL) > 0 {
			break
		}
	}

	// set orderLabels but notdfind order href, return nil to use web driver to get page again
	if len(orderLabels) > 0 {
		if len(orderURL) <= 0 {  // can not get href
			log.Info("can not find order href, use web driver to get page again")

			return nil, nil
		} else {
			if !strings.Contains(orderURL, "/") {  // href not a path, not legal
				log.Info("can not find order href, use web driver to get page again")

				return nil, nil
			}
		}
	}

	if len(orderURL) <= 0 {
		log.Info("do not need order page by requestDocHTTP")

		return doc, nil
	}

	// request order page doc only if get order url success
	orderDoc := t.httpService.GetDocRequestGet(orderURL)
	if orderDoc == nil {
		log.Error("request get order doc failed by requestDocHTTP")

		return doc, nil
	}

	return doc, orderDoc
}


// checkResLegal returns true if the resource is legal
func checkResLegal(pi *cm.ProInfo) bool {
	if pi != nil && len(pi.Cover) > 0 && len(pi.Desc) > 0 {
		return true
	} else {
		return false
	}
}

// parseWebPage for parse web page of this pageURL to get site resource
func (t *TaskService) parseWebPage(pageURL string) *cm.ProInfo {
	u, _ := url.Parse(pageURL)
	domainMD5 := ut.GetMD5(u.Host)

	// debug
	log.WithFields(log.Fields{
		"domain":		u.Host,
		"domainMD5":	domainMD5,
		"pageURL":		pageURL,
	}).Debug("enter TaskParseURL request get")

	var pi *cm.ProInfo
	value, ok := t.site.SitesLabelMaps.Load(domainMD5)
	if ok {
		labels := value.(*cm.LabelsParse)

		// debug
		log.WithFields(log.Fields{
			"character":	labels.Character,
			"order":		labels.Order,
		}).Debug("enter TaskParseURL request get")

		if labels.Character == cm.HTMLFormat {  // use html template to parse
			doc, orderDoc := t.requestDocHTTP(pageURL, labels.Order)

			// ******************************* debug **************************
			var docHTML string
			var orderHTML string
			if doc != nil {
				docHTML,_ = doc.Html()
			}
			if orderDoc != nil {
				orderHTML, _ = orderDoc.Html()
			}
			log.WithFields(log.Fields{
				"order":		labels.Order,
				"mainDoc":		docHTML,
				"orderDoc":		orderHTML,
			}).Debug("get page html by requestDocHTTP")
			// ********************************************************************

			if doc != nil {
				pi = t.site.ParseInfoCommonHTML(pageURL, doc, orderDoc, labels)
				if checkResLegal(pi) {
					t.ResChan <- pi

					return pi
				}
			}
		} else if labels.Character == cm.JSONFormat {  // use json template to parse
			jsonByte := t.httpService.GetJsonRequestPost(pageURL)

			// debug
			log.WithFields(log.Fields{
				"json":	string(jsonByte),
			}).Debug("get page json by GetJsonRequestPost")

			if jsonByte != nil {
				pi = t.site.ParseInfoCommonJSON(pageURL, jsonByte, labels)
				if checkResLegal(pi) {
					t.ResChan <- pi

					return pi
				}
			}
		}
	}

	// get doc by web driver if can not parse above
	currentURL, doc, orderDoc := t.requestDocWebDriver(pageURL)
	if doc == nil {
		return nil
	}
	u, _ = url.Parse(currentURL)
	domainMD5 = ut.GetMD5(u.Host)
	value, ok = t.site.SitesLabelMaps.Load(domainMD5)
	labels := value.(*cm.LabelsParse)

	// ******************************* debug **************************
	var docHTML string
	var orderHTML string
	if doc != nil {
		docHTML,_ = doc.Html()
	}
	if orderDoc != nil {
		orderHTML, _ = orderDoc.Html()
	}
	log.WithFields(log.Fields{
		"domain":		u.Host,
		"domainMD5":	domainMD5,
		"pageURL":		pageURL,
		"order":		labels.Order,
		"mainDoc":		docHTML,
		"orderDoc":		orderHTML,
	}).Debug("enter web driver")
	// ********************************************************************

	if ok {
		pi = t.site.ParseInfoCommonHTML(currentURL, doc, orderDoc, labels)
		if checkResLegal(pi) {
			t.ResChan <- pi

			return pi
		}
	}

	// debug
	if pi == nil {
		log.Debug("parse failed, can not create ProInfo instance")
	}
	if len(pi.Cover) <= 0 {
		log.Debug("parse failed, can not get cover")
	}
	if len(pi.Desc) <= 0 {
		log.Debug("parse failed, can not get desc")
	}
	log.WithFields(log.Fields{
		"domain":	u.Host,
		"pageURL":	pageURL,
	}).Debug("parse failed ！！！")

	return nil
}

// chDirMod for change file's mod
func chDirMod(dir string) {
	log.WithFields(log.Fields{
		"dir":	dir,
	}).Info("all resources of this web page have been downloaded, chmod 755")
	exec.Command("chmod", "-R", "755", dir).Run()
}


// saveProInfo for save site resource info
func (t *TaskService) saveProInfo(ce *cm.CargoExtInfo, pi *cm.ProInfo) {
	// 先删除后插入
	itemMaterials := new(md.WcCargoMaterials)
	itemMaterials.CargoId = ce.CargoID
	t.db.Delete(itemMaterials, "wc_cargo_materials")

	itemReptile := new(md.WcCargoReptileTask)
	itemReptile.CargoId = ce.CargoID
	t.db.Delete(itemReptile, "wc_cargo_reptile_task")

	var cover string
	if len(pi.Cover) > 0 {
		cover = ut.ToJson(pi.Cover)
	}

	if strings.Contains(pi.Desc, "script") {
		//pi.Desc = strings.ReplaceAll(pi.Desc, "script", "--")
		doc, _ := goquery.NewDocumentFromReader(strings.NewReader(pi.Desc))
		doc.Find("script").Remove()
		doc.Find("head").Remove()
		pi.Desc, _ = doc.Html()
		pi.Desc = strings.ReplaceAll(pi.Desc, "<html>", "<div>")
		pi.Desc = strings.ReplaceAll(pi.Desc, "</html>", "</div>")
	}

	// insert materials
	itemMaterials.CargoId = ce.CargoID
	itemMaterials.CargoExtId = uint(ce.ID)
	itemMaterials.PageLink = ce.LandingURL
	itemMaterials.Cover = cover
	//itemMaterials.Images = images
	itemMaterials.Content = pi.Desc
	ctime := time.Now().Unix()
	strInt64 := strconv.FormatInt(ctime, 10)
	ctime16 ,_ := strconv.Atoi(strInt64)
	itemMaterials.CreateTime = ctime16
	t.db.SingleInsert(itemMaterials)

	// insert reptile
	itemReptile.CargoId = ce.CargoID
	itemReptile.CargoExtId = uint(ce.ID)
	itemReptile.Type = 0
	itemReptile.ReptileId = 0
	itemReptile.PageLink = ce.LandingURL
	itemReptile.Status = 2
	itemReptile.AdminId = 1
	itemReptile.CreateTime = ctime16
	t.db.SingleInsert(itemReptile)

	// chmod
	//fp := pi.Cover[0]
	//dir := strings.ReplaceAll(fp[:strings.LastIndex(fp, "/") + 1], cm.ImagePrefixUploads, cm.ImageDir)
	//chDirMod(dir)

	// only for debug
	log.WithFields(log.Fields{
		"itemMaterials":	itemMaterials,
		"itemReptile":		itemReptile,
	}).Debug("saveProInfo by single insert")
}

// getImageInfoURL return image's url
func getImageInfoURL(images []cm.ImageInfo) []string {
	var path []string
	for _, image := range images {
		//index := strings.Index(image.URL, "/")
		//imagePath := image.RootPath + image.URL[index :]
		//if len(imagePath) <= len(image.RootPath) {
		//	continue
		//}

		path = append(path, image.URL)
	}

	return path
}

// deSerializationImageInfo return pointer of list ImageInfo
func deSerializationImageInfo(record []byte) *[]cm.ImageInfo {
	var images []cm.ImageInfo
	if err := jsoniter.Unmarshal(record, &images); err != nil{
		log.WithFields(log.Fields{
			"error":	err.Error(),
		}).Error("failed to unmarshal this data to images struct by deSerializationImageInfo")

		return nil
	}

	return &images
}

// getSpecifiedRes return list of specified site resource
func getSpecifiedRes(pi *cm.ProInfo, resTitle string) interface{} {
	switch resTitle {
		case "cover":
			return pi.Cover

		case "title":
			return pi.Title

		case "currency":
			return pi.Currency

		case "price":
			return pi.Price

		case "desc":
			return pi.Desc

		case "good":
			return pi.Good

		case "spec":
			return pi.Spec


		default:
			return ut.ToJson(pi)
	}
}

// queryResource for query specified site resource by url md5, return true if query success
func (t *TaskService) QueryResource(pageURL string, resTitle string) interface{} {
	if len(pageURL) <= 0 {
		log.Error("do not get page url by queryResource")

		return ""
	}

	// debug
	ok := false
	num := 0

	//items := new([]orm.Params)
	//// set query condition
	//cond := orm.NewCondition()
	//cond = cond.And("url_md5", ut.GetMD5(pageURL))
	//num, ok := t.db.QueryColumn(items, "wc_cargo_materials", cond, "-create_time", resTitle, 0)
	if !ok || num <= 0 {
		log.WithFields(log.Fields{
			"pageURL":	pageURL,
		}).Info("can not find site resource of this page url, start to parse page by queryResource")

		pi := t.parseWebPage(pageURL)
		if pi == nil {
			log.WithFields(log.Fields{
				"pageURL":	pageURL,
			}).Error("parse web page failed by queryResource")

			return ""
		}

		return getSpecifiedRes(pi, resTitle)
	}

	//// return resource from db
	//if resTitle == "cover" {
	//	imageInfo := (*items)[0]["cover"].([]cm.ImageInfo)
	//
	//	return parseImagePath(imageInfo)
	//}
	//
	//v := (*items)[0][resTitle].(string)

	return ""
}