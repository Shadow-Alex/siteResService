/*
  Package sites for parse site template
*/

package sites

import (
	"os"
	"io"
	"fmt"
	"regexp"
	"strconv"
	"time"
	"sync"
	"strings"
	"net/url"
	"io/ioutil"
	"encoding/csv"

	"github.com/astaxie/beego"
	"github.com/PuerkitoBio/goquery"
	jsoniter "github.com/json-iterator/go"
	log "github.com/sirupsen/logrus"

	sc "../../scheduler"
	hs "../../httpService"
	cm "../../common"
	ut "../../util"
)

type SiteService struct {
	downloadChan 	chan cm.DownLoadInfo  // for download resource
	http			*hs.ServiceHTTP
	scheduler		*sc.Scheduler
	SitesLabelMaps	sync.Map  // sites label maps
}

var instance *SiteService
var initTaskOnce sync.Once
var rootPath string
var setKVMatch *regexp.Regexp

// GetSiteServiceInstance return siteService pointer instance
func GetSiteServiceInstance() *SiteService {
	initTaskOnce.Do(func() {
		instance = new(SiteService)
		instance.init()

		go instance.dispatch()

		log.Info("init site service instance success...")
	})

	return instance
}

// init for instance
func (s *SiteService) init () {
	var err error
	rootPath, err = os.Getwd()
	if err != nil {
		rootPath = cm.ImagePrefixDefault

		log.WithFields(log.Fields{
			"rootPath":	rootPath,
		}).Error("get pwd path failed by siteService init, use default")
	}

	size := beego.AppConfig.DefaultInt("channelSize", cm.MaxChannelSize)
	s.downloadChan = make(chan cm.DownLoadInfo, size)
	s.http = hs.GetHTTPInstance()
	s.scheduler = sc.GetScheduler()
	s.initSitesLabelMaps()

	// regexp for get style and value
	setKVMatch = regexp.MustCompile(`\{(.*)\}`)
}

// note !!!
// each label in labels struct support find cascade html label (cascade json label) that to get accurate result
// for html, each cascade label use "|" to separate single labels,
// and html will use labels in "Order" to get order page href to request order page html, "|" usage for example:
// ".col-md-6|.title" means to find value in <div class="col-md-6"><div class="title">value</div></div>
// for web driver, the usage of separate "|" is the same as html,
// and web driver will use labels in "Order" to do click and redirect to order page;
// for json, the usage of separate "|" is the same as html,
// and use ";" to indicates that the previous layer is list, ";" only use for cover, title, price, and only use once, for example:
// "data|products|covers;name" means to get value in date:{product:{covers:[name:value,name:value]}}
// addSiteResource for add site resource into SitesLabelMaps templates,
// the sequence of params []string : domain,character,order,cover,title,price,desc,spec,set,pageURL,type
func (s *SiteService) addSiteResource(record []string) {
	domain := record[0]
	if len(domain) <= 0 {
		log.Panic("can not get domain str")
	}

	domainMD5 := ut.GetMD5(domain)
	charLabel := record[1]
	orderLabels := strings.Split(record[2], "+")
	// to avoid split empty field of record and return a list which len = 1 !!!
	// and avoid "" string
	if len(record[2]) <= 0 || record[2] == "" {
		orderLabels = []string{}
	}
	coverLabels := strings.Split(record[3], "+")
	if len(record[3]) <= 0 || record[3] == "" {
		coverLabels = []string{}
	}
	titleLabels := strings.Split(record[4], "+")
	if len(record[4]) <= 0 || record[4] == "" {
		titleLabels = []string{}
	}
	priceLabels := strings.Split(record[5], "+")
	if len(record[5]) <= 0 || record[5] == "" {
		priceLabels = []string{}
	}
	descLabels := strings.Split(record[6], "+")
	if len(record[6]) <= 0 || record[6] == "" {
		descLabels = []string{}
	}
	specLabels := strings.Split(record[7], "+")
	if len(record[7]) <= 0 || record[7] == "" {
		specLabels = []string{}
	}
	setLabels := strings.Split(record[8], "+")
	if len(record[8]) <= 0 || record[8] == "" {
		setLabels = []string{}
	}

	value, ok := s.SitesLabelMaps.Load(domainMD5)
	if ok {  // insert new labels to exist domain template
		lab := value.(*cm.LabelsParse)
		lab.Character = charLabel
		lab.Order = orderLabels
		for _, cover := range coverLabels {
			lab.Cover = append(lab.Cover, cover)
		}
		for _, title := range titleLabels {
			lab.Title = append(lab.Title, title)
		}
		for _, price := range priceLabels {
			lab.Price = append(lab.Price, price)
		}
		for _, desc := range descLabels {
			lab.Desc = append(lab.Desc, desc)
		}
		for _, spec := range specLabels {
			lab.Spec = append(lab.Spec, spec)
		}
		for _, set := range setLabels {
			lab.Set = append(lab.Set, set)
		}

		return
	}

	// add new domain template
	labels := &cm.LabelsParse{
		Character:	charLabel,
		Order:		orderLabels,
		Cover:		coverLabels,
		Title:		titleLabels,
		Price:		priceLabels,
		Desc:		descLabels,
		Spec:		specLabels,
		Set:		setLabels,
	}
	s.SitesLabelMaps.Store(domainMD5, labels)
}

func (s *SiteService) initSitesLabelMaps() {
	dat, err := ioutil.ReadFile("./conf/templateResource.csv")
	if err != nil {
		log.WithFields(log.Fields{
			"path":		"./conf/templateResource.csv",
			"error":	err.Error(),
		}).Error("read templateResource file")

		return
	}

	r := csv.NewReader(strings.NewReader(string(dat)))
	for {
		record, err := r.Read()
		if err == io.EOF {
			log.Info("finish read all template...")

			break
		}

		if err != nil {
			log.WithFields(log.Fields{
				"path":		"./conf/templateResource.csv",
				"error":	err.Error(),
			}).Panic("read templateResource line failed")

			continue
		}

		if len(record) < 11 {
			log.WithFields(log.Fields{
				"record":	record,
			}).Error("can not use this template, due to insufficient character")
		}

		s.addSiteResource(record)

		// debug
		domain := record[0]
		domainMD5 := ut.GetMD5(domain)
		value, _ := s.SitesLabelMaps.Load(domainMD5)
		labels := value.(*cm.LabelsParse)
		log.WithFields(log.Fields{
			"url":		record[9],
			"domain":	domain,
			"labels":	*labels,
		}).Debug("template")
	}
}

// testFunc only for test, do not use this func
func (s *SiteService) testFunc() {
	// java script, https://www.yuanddd.com/tzbi3?a=twwj0113&c=f&b=john
	//labels := &cm.LabelsParse{
	//	Character:	"web",
	//	Order:		[]string{`div[class=submit-btn-cont]`},
	//	Cover:		[]string{`.el-carousel__item`},
	//	Title:		[]string{`.time-up-text`},
	//	Price:		[]string{`.time-up-title`},
	//	Desc:		[]string{`#goods-detail`},
	//	Spec:		[]string{`.time-up-text`},
	//	Set:		[]string{`.select-size`},
	//}
	// the sequence of params []string : domain,character,order,cover,title,price,desc,spec,set,pageURL,type
	s.addSiteResource([]string{"www.yuanddd.com","web","div[class=submit-btn-cont]",".el-carousel__item",
			".time-up-text",".time-up-title","#goods-detail",".time-up-text",".select-size",
			"https://www.yuanddd.com/tzbi3?a=twwj0113&c=f&b=john","java script"})
	// http://wangbada.com/detail/CZLR15AS1H.html
	//labels = &cm.LabelsParse{
	//	Character:	"html",
	//	Order:		[]string{`.foot-nav-2|a`},
	//	Cover:		[]string{`.box-image`},
	//	Title:		[]string{`.title|h1`},
	//	Price:		[]string{`.price`},
	//	Desc:		[]string{`.box-content`},
	//	Spec:		[]string{`#big_1|.con_ul|.con`},
	//	Set:		[]string{`.rows-id-params-select`},
	//}
	s.addSiteResource([]string{"wangbada.com","html",".foot-nav-2|a",".box-image",
		".title|h1",".price",".box-content","#big_1|.con_ul|.con",".rows-id-params-select",
		"http://wangbada.com/detail/CZLR15AS1H.html","normal html"})
	// https://www.playbyplay.com.tw/product/detail/391860
	s.addSiteResource([]string{"www.playbyplay.com.tw","html","",".swiper-wrapper",
		".mobile_product_info",".product_description|.product_price|.js_onsale_price|.font_montserrat",
		".product_feature",".form_collection","","https://www.playbyplay.com.tw/product/detail/391860","nomal html"})
	// https://rkw.magelet.com/p/WZJD_281
	s.addSiteResource([]string{"rkw.magelet.com","html","",".swiper-wrapper",
		"#buy|span",".price-l",".content",".normsArr","#radio",
		"https://rkw.magelet.com/p/WZJD_281","normal html"})
	// POST to get json, https://www.kelmall.com/p/dtys
	s.addSiteResource([]string{"www.kelmall.com","json","","data|products|covers;imgurl",
		"data|products|name","data|products|combos;name","data|products|content|detail","data|products|attribute",
		"data|products|combos","https://www.kelmall.com/p/dtys","post json"})
	// https://ui.cxet2bn.com/index.php/products/detail/sn/wq30ssyy
	s.addSiteResource([]string{"ui.cxet2bn.com","html","#single_right_now",".swiper-slide-active",
		".title|span",".goods_price",".comments","#data_foreach1|.compose_select;.con_ul;",
		".rows-id-params-select|.rows-params|.alizi-params",
		"https://ui.cxet2bn.com/index.php/products/detail/sn/wq30ssyy","normal html"})
	// url relocation, use web driver, https://lihi1.cc/P5IqC -> https://9j2b1b.1shop.tw/kkmdun
	s.addSiteResource([]string{"9j2b1b.1shop.tw","web","",".col-3",
		".container|span",".action-content",".customize|section","",".action-content",
		"https://lihi1.cc/P5IqC","url relocation"})
	// this host exist above, https://www.playbyplay.com.tw/product/detail/448127
	//s.addSiteResource([]string{"www.playbyplay.com.tw", "html", ".col-3", ".discount", ".sale", ".customize", "", ".action-content"})
	// url relocation, use web driver, https://reurl.cc/NaK17Q -> https://www.huangjun.tw/products/%E9%9F%93%E7%89%88%E4%BF%AE%E8%BA%AB%E5%90%8A%E5%B8%B6%E5%B0%8F%E8%83%8C%E5%BF%83
	s.addSiteResource([]string{"www.huangjun.tw","web","",".col-md-6|.ng-scope",
		".add-to-cart|.title",".not-same-price|.price-sale",".description-container",".ng-touched","",
		"https://reurl.cc/NaK17Q","url relocation"})
	// java script, use web driver, https://fishs168.com.tw/product/detail/447463
	s.addSiteResource([]string{"fishs168.com.tw","html","",".swiper-container",
		".mobile_product_info",".js_onsale_price|.font_montserrat",".product_feature",".mobile_product_info","",
		"https://fishs168.com.tw/product/detail/447463","java script"})
	//// https://www.ogmax.net/products/256, java script
	//// https://www.easyshop7.com/tw/p/aQZ3qa
	//s.addSiteResource("www.easyshop7.com", "html", ".swiper-slide", ".reveal_title", ".price", ".content", ".sys_spec_text", "")
	//// https://hooetw.com/detail/SS2020.html
	//s.addSiteResource("hooetw.com", "html", ".box-image", ".title", ".price", "#detial-context", "#big_1", ".rows-id-params-select")
	//// http://biishopping.com/detail/BBBNNY.html, java script
	//// TODO: https://e8vyco.1shop.tw/gg30pl, java script, all images
	//// https://rakuma.rakuten.com.tw/item/f0000031575651301975, java script
	//// https://evabathboutique.1shop.tw/classic?tag=pp, java script
}

// GetDocWebDriver returns pointer of goquery.Document instance
func (s *SiteService) GetDocWebDriver(pageURL string) *goquery.Document {
	resp := s.http.RequestGet(pageURL, hs.DefaultHeader())
	if resp == nil {
		return nil
	}
	if resp.StatusCode != 200 {
		log.WithFields(log.Fields{
			"statusCode":	resp.StatusCode,
			"body":			string(resp.Body),
		}).Error("request html error")

		return nil
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(resp.Body)))
	if err != nil {
		log.Error(err.Error())
		return nil
	}

	return doc
}

// GetResourceUrl returns image url
func GetResourceURL(imageSrc string, pageURL string) string {
	u, _ := url.Parse(pageURL)
	domain := u.Scheme + "://" + u.Host

	imageURL := imageSrc

	if len(imageURL) < 6 {
		return ""
	}

	if imageURL[0:4] == "http" {
		// do none
	} else if imageURL[0:2] == "//" {
		imageURL = "http:" + imageSrc
	} else if imageURL[0:1] == "/" {
		imageURL = domain + imageSrc
	} else {
		imageURL = pageURL + imageSrc
	}

	imageURL = strings.ReplaceAll(imageURL, "\r", "")
	imageURL = strings.ReplaceAll(imageURL, "\n", "")

	if strings.Contains(imageURL, "?") {
		imageURL = imageURL[:strings.Index(imageURL, "?")]
	}

	return imageURL
}

// NewImage returns ImageInfo instance
func NewImage(imageURL string, imageName string, imageDir string, isCover bool) cm.ImageInfo {
	var image cm.ImageInfo
	image.Title = imageName
	image.URL = imageURL
	image.Iid = imageName[:strings.Index(imageName, ".")]
	image.Gid = ""
	image.Original = image.Title
	image.Thumb = strings.ReplaceAll(imageDir+"/"+imageName, cm.ImageDir, cm.ImagePrefixUploads)
	image.Md5 = image.Iid
	image.RootPath = rootPath
	image.IsCover = isCover
	image.IsImage = false

	return image
}

// generateImageName return md5 image name
func generateImageName(url string, rename string) string {
	imageName := rename
	if rename == "" {
		var suffix string
		index := strings.LastIndex(url, ".")
		if index >= 0 {
			suffix = url[index :]
		}

		if suffix == ".gif" {
			suffix = ".gif"
		} else if suffix == ".png" {
			suffix = ".png"
		} else if suffix == ".mp4" {
			suffix = ".mp4"
		} else {
			suffix = ".jpg"
		}

		imageName = fmt.Sprint(ut.GetMD5(url), suffix)
	}

	return imageName
}

// ReplaceImagePaths returns description of image
func (s *SiteService) ReplaceImagePaths(desc string, pageURL string, imageDir string) string {
	resourceMap := make(map[string]string)
	docDesc, _ := goquery.NewDocumentFromReader(strings.NewReader(desc))
	docDesc.Find("img").Each(func(i int, selection *goquery.Selection) {
		imageSrc, ok := selection.Attr("data-original")
		if !ok || len(imageSrc) <= 0 {
			imageSrc, ok = selection.Attr("src")
			if !ok || len(imageSrc) <= 0 {
				return
			}
		}

		imageURL := GetResourceURL(imageSrc, pageURL)
		imageName := generateImageName(imageURL, "")
		downloadInfo := cm.DownLoadInfo{
			URL:	imageURL,
			Name:	imageName,
		}
		s.downloadChan <- downloadInfo  // push to download task, asynchronous download
		//imageName, okD := s.http.DownloadImage(imageURL, imageDir, "")
		resourceMap[imageSrc] = imageDir + "/" + imageName
	})

	docDesc.Find("video").Each(func(i int, selection *goquery.Selection) {
		videoSrc, okS := selection.Attr("src")
		if okS && len(videoSrc) > 0 {
			vURL := GetResourceURL(videoSrc, pageURL)
			imageName := generateImageName(vURL, "")
			downloadInfo := cm.DownLoadInfo{
				URL:	vURL,
				Name:	imageName,
			}
			s.downloadChan <- downloadInfo  // push to download task, asynchronous download
			//imageName, okD := s.http.DownloadImage(vURL, imageDir, "")
			resourceMap[videoSrc] = imageDir + "/" + imageName
		}

		posterSrc, okP := selection.Attr("poster")
		if okP && len(posterSrc) > 0 {
			imgURL := GetResourceURL(posterSrc, pageURL)
			imageName := generateImageName(imgURL, "")
			downloadInfo := cm.DownLoadInfo{
				URL:	imgURL,
				Name:	imageName,
			}
			s.downloadChan <- downloadInfo  // push to download task, asynchronous download
			//imageName, okD := s.http.DownloadImage(imgURL, imageDir, "")
			resourceMap[posterSrc] = imageDir + "/" + imageName
		}

		selection.Find("source").Each(func(j int, selection *goquery.Selection) {
			videoSrc, ok := selection.Attr("src")
			if ok && len(videoSrc) > 0 {
				vURL := GetResourceURL(videoSrc, pageURL)
				imageName := generateImageName(vURL, "")
				downloadInfo := cm.DownLoadInfo{
					URL:	vURL,
					Name:	imageName,
				}
				s.downloadChan <- downloadInfo  // push to download task, asynchronous download
				//imageName, okD := s.http.DownloadImage(vURL, imageDir, "")
				resourceMap[videoSrc] = imageDir + "/" + imageName
			}
		})
	})

	for k, v := range resourceMap {
		//v = strings.ReplaceAll(v, cm.ImageDir, cm.ImagePrefixDefault)
		desc = strings.ReplaceAll(desc, k, v)
	}

	//logs.Info(desc)
	return desc
}

// checkSelectionLegal return true if selection is not empty
func checkSelectionLegal(selection interface{}, mode string, labels string, funcName string) bool {
	var body string
	if mode == "html" {
		body, _ = selection.(*goquery.Selection).Html()
	} else if mode == "json" {
		body = selection.(jsoniter.Any).ToString()
	}

	if len(body) <= 0 {
		log.WithFields(log.Fields{
			"labels":	labels,
		}).Error("failed to get body of labels by ", funcName)

		return false
	}


	return true
}

// parseCoverImages returns list of ImageInfo instance
func (s *SiteService) parseCoverImages(selection *goquery.Selection, pageURL string, imageDir string) []cm.ImageInfo {
	var images []cm.ImageInfo
	selection.Find("img").Each(func(i int, selc *goquery.Selection) {
		imageSrc, ok := selc.Attr("src")
		if !ok || len(imageSrc) <= 0 {
			imageSrc, ok = selc.Attr("data-src")
			if !ok || len(imageSrc) <= 0 {
				imageSrc, ok = selc.Attr("data-original")
				if !ok || len(imageSrc) <= 0 {
					return
				}
			}
		}

		// download image
		imageURL := GetResourceURL(imageSrc, pageURL)
		imageName := generateImageName(imageURL, "")
		downloadInfo := cm.DownLoadInfo{
			URL:	imageURL,
			Name:	imageName,
		}
		s.downloadChan <- downloadInfo  // push to download task, asynchronous download
		//imageName, ok := s.http.DownloadImage(imageURL, imageDir, "")
		images = append(images, NewImage(imageURL, imageName, imageDir, true))
	})

	return images
}

// ParseCoverImagesHTML parse by html, returns list of cover ImageInfo instance
func (s *SiteService) ParseCoverImagesHTML(doc *goquery.Document, pageURL string, imageDir string, selectors []string) []cm.ImageInfo {
	var images []cm.ImageInfo
	for _, selector := range selectors {
		labels := strings.Split(selector, cm.LabelSeparate)
		selection := doc.Find(labels[0])
		for i := 1; i < len(labels); i++ {  // cascade find label
			selection = selection.Find(labels[i])
		}

		if !checkSelectionLegal(selection, "html", selector, "ParseCoverImagesHTML") {
			continue
		}

		images = s.parseCoverImages(selection, pageURL, imageDir)
		if len(images) > 0 {
			//  debug
			log.WithFields(log.Fields{
				"coverLabel":	selector,
			}).Debug("ParseCoverImages success")

			return images
		}
	}

	//  debug
	log.WithFields(log.Fields{
		"coverLabel":	selectors,
	}).Debug("can not get cover by ParseCoverImages")

	return images
}

// ParseCoverImagesJSON parse by json, returns list of cover ImageInfo instance
func (s *SiteService) ParseCoverImagesJSON(body []byte, pageURL string, imageDir string, selectors []string) []cm.ImageInfo {
	var images []cm.ImageInfo
	for _, selector := range selectors {
		labelList := strings.Split(selector, cm.ListSeparate)
		labels := strings.Split(labelList[0], cm.LabelSeparate)
		labelsNum := len(labels)
		jsonIter := jsoniter.Get(body, labels[0])
		for i := 1; i < labelsNum; i++ {  // cascade find label
			jsonIter = jsonIter.Get(labels[i])
		}

		if !checkSelectionLegal(jsonIter, "json", selector, "ParseCoverImagesJSON") {
			continue
		}

		var imageURL string
		if len(labelList) > 1 { // if contains "#", means labelList[1] indicates that the previous layer is list
			iterNum := jsonIter.Size()
			for i := 0; i < iterNum; i++ {
				imageURL = jsonIter.Get(i).Get(labelList[1]).ToString()
				imageURL = GetResourceURL(imageURL, pageURL)
				imageName := generateImageName(imageURL, "")
				downloadInfo := cm.DownLoadInfo{
					URL:	imageURL,
					Name:	imageName,
				}
				s.downloadChan <- downloadInfo  // push to download task, asynchronous download
				//imageName, ok := s.http.DownloadImage(imageURL, imageDir, "")
				images = append(images, NewImage(imageURL, imageName, imageDir, true))
			}
		} else {
			imageURL = jsonIter.ToString()  // if do not contains "#" means do not has list layer
			imageURL = GetResourceURL(imageURL, pageURL)
			imageName := generateImageName(imageURL, "")
			downloadInfo := cm.DownLoadInfo{
				URL:	imageURL,
				Name:	imageName,
			}
			s.downloadChan <- downloadInfo  // push to download task, asynchronous download
			//imageName, ok := s.http.DownloadImage(imageURL, imageDir, "")
			images = append(images, NewImage(imageURL, imageName, imageDir, true))
		}

		if len(images) > 0 {
			//  debug
			log.WithFields(log.Fields{
				"coverLabel":	selector,
			}).Debug("ParseCoverImagesJSON success")

			return images
		}
	}

	//  debug
	log.WithFields(log.Fields{
		"coverLabel":	selectors,
	}).Debug("can not get cover by ParseCoverImagesJSON")

	return images
}

// ParseTitleHTML parse by html, returns title after parse
func (s *SiteService) ParseTitleHTML(doc *goquery.Document, selectors []string) string {
	for _, selector := range selectors {
		labels := strings.Split(selector, cm.LabelSeparate)
		selection := doc.Find(labels[0])
		for i := 1; i < len(labels); i++ {  // cascade find label
			selection = selection.Find(labels[i])
		}

		if !checkSelectionLegal(selection, "html", selector, "ParseTitleHTML") {
			continue
		}

		title := selection.Text()
		if len(title) > 0 {
			//  debug
			log.WithFields(log.Fields{
				"titleLabel":	selector,
			}).Debug("ParseTitle success")

			return title
		}
	}

	//  debug
	log.WithFields(log.Fields{
		"titleLabel":	selectors,
	}).Debug("can not get title by ParseTitleHTML")

	return ""
}

// ParseTitleJSON parse by json, returns title after parse
func (s *SiteService) ParseTitleJSON(body []byte, selectors []string) string {
	for _, selector := range selectors {
		labelList := strings.Split(selector, cm.ListSeparate)
		labels := strings.Split(labelList[0], cm.LabelSeparate)
		labelsNum := len(labels)
		jsonIter := jsoniter.Get(body, labels[0])
		for i := 1; i < labelsNum; i++ {  // cascade find label
			jsonIter = jsonIter.Get(labels[i])
		}

		if !checkSelectionLegal(jsonIter, "json", selector, "ParseTitleJSON") {
			continue
		}

		var finalTitle string  // final title
		var titleList []string  // if contains multi title
		if len(labelList) > 1 {  // if contains "#", means labelList[1] indicates that the previous layer is list
			iterNum := jsonIter.Size()
			for i := 0; i < iterNum; i++ {
				titleList = append(titleList, jsonIter.Get(i).Get(labelList[1]).ToString())
			}
			finalTitle = strings.Join(titleList, ",")  // "," for separate each title
		} else {
			finalTitle = jsonIter.ToString()
		}

		if len(finalTitle) > 0 {
			//  debug
			log.WithFields(log.Fields{
				"titleLabel":	selector,
			}).Debug("ParseTitleJSON success")

			return finalTitle
		}
	}

	//  debug
	log.WithFields(log.Fields{
		"titleLabel":	selectors,
	}).Debug("can not get title by ParseTitleJSON")

	return ""
}

// ParsePriceHTML parse by html, returns price after parse
func (s *SiteService) ParsePriceHTML(doc *goquery.Document, selectors []string) string {
	for _, selector := range selectors {
		labels := strings.Split(selector, cm.LabelSeparate)
		selection := doc.Find(labels[0])
		for i := 1; i < len(labels); i++ {  // cascade find label
			selection = selection.Find(labels[i])
		}

		if !checkSelectionLegal(selection, "html", selector, "ParsePriceHTML") {
			continue
		}

		str := selection.Text()
		str = strings.ReplaceAll(str, "\n", "")
		str = strings.ReplaceAll(str, "\t", "")
		str = strings.ReplaceAll(str, "\r", "")
		str = strings.TrimSpace(str)
		if len(str) > 0 {
			//  debug
			log.WithFields(log.Fields{
				"priceLabel":	selector,
			}).Debug("ParsePriceHTML success")

			return str
		}
	}

	//  debug
	log.WithFields(log.Fields{
		"priceLabel":	selectors,
	}).Debug("can not get price by ParsePriceHTML")

	return ""
}

// ParsePriceJSON parse by json, returns price after parse
func (s *SiteService) ParsePriceJSON(body []byte, selectors []string) string {
	for _, selector := range selectors {
		labelList := strings.Split(selector, cm.ListSeparate)
		labels := strings.Split(labelList[0], cm.LabelSeparate)
		labelsNum := len(labels)
		jsonIter := jsoniter.Get(body, labels[0])
		for i := 1; i < labelsNum; i++ {  // cascade find label
			jsonIter = jsonIter.Get(labels[i])
		}

		if !checkSelectionLegal(jsonIter, "json", selector, "ParsePriceJSON") {
			continue
		}

		var finalPrice string  // final price
		var priceList []string  // if contains multi price
		if len(labelList) > 1 {  // if contains "#", means labelList[1] indicates that the previous layer is list
			iterNum := jsonIter.Size()
			for i := 0; i < iterNum; i++ {
				priceList = append(priceList, jsonIter.Get(i).Get(labelList[1]).ToString())
			}
			finalPrice = strings.Join(priceList, ",")  // "," for separate each price
		} else {
			finalPrice = jsonIter.ToString()
		}

		finalPrice = strings.ReplaceAll(finalPrice, "\n", "")
		finalPrice = strings.ReplaceAll(finalPrice, "\t", "")
		finalPrice = strings.ReplaceAll(finalPrice, "\r", "")
		finalPrice = strings.TrimSpace(finalPrice)
		if len(finalPrice) > 0 {
			//  debug
			log.WithFields(log.Fields{
				"priceLabel":	selector,
			}).Debug("ParsePriceJSON success")

			return finalPrice
		}
	}

	//  debug
	log.WithFields(log.Fields{
		"priceLabel":	selectors,
	}).Debug("can not get price by ParsePriceJSON")

	return ""
}

// iterativeHTML for iterative locate selection return located selection
func iterativeHTML(selection *goquery.Selection, selector string) *goquery.Selection {
	labels := strings.Split(selector, cm.LabelSeparate)
	for i := 1; i < len(labels); i++ {  // cascade find label
		selection = selection.Find(labels[i])
	}

	return selection
}

// iterativeHTML for iterative locate selection return located selection
func iterativeLoopHTML(selection *goquery.Selection, selector string, levels []string, level int, urlMD5 string, style string, dataList *[][]string) *[][]string {
	index := strings.Index(selector, cm.ListSeparate)
	if index == -1 {
		selection = iterativeHTML(selection, selector)
		selection.Each(func(i int, selc *goquery.Selection) {
			html, err := selc.Html()
			if err != nil {
				log.Error("can not get html by iterativeLoopHTML")
				return
			}
			var data []string
			data = append(data, urlMD5)
			data = append(data, style)
			data = append(data, strings.Join(levels, "-"))
			data = append(data, html)
			*dataList = append(*dataList, data)
		})

		return dataList
	}

	// get style and its labels
	sub := setKVMatch.FindAllString(selector[: index], 1)
	if len(sub) > 0 {
		values := strings.Split(sub[0], ":")
		style = selection.Find(values[0]).Text()  // get style
		selection = selection.Find(values[1])  // get style's html
	}

	selection = iterativeHTML(selection, selector[: index])  // get first
	if !checkSelectionLegal(selection, "html", selector, "parseHTMLStrImage") {
		return dataList
	}

	selection.Each(func(i int, selc *goquery.Selection) {
		levels = append(levels, "level-" + strconv.Itoa(level))
		dataList = iterativeLoopHTML(selc, selector[index + 1 :], levels, level + 1, urlMD5, style, dataList) // process rest, and +1 to avoid cm.ListSeparate
	})

	return dataList
}

// parseHTMLDesc for get desc html string and download image by parse doc, return html
func (s *SiteService) parseHTMLDesc(doc *goquery.Document, pageURL string, imageDir string, selectors []string) string {
	var html string

	for _, selector := range selectors {
		selection := iterativeHTML(doc.Selection, selector)  // get first
		if !checkSelectionLegal(selection, "html", selector, "parseHTMLStrImage") {
			continue
		}

		html, _ = selection.Html()
		if len(html) > 0 {
			html = s.ReplaceImagePaths(html, pageURL, imageDir)

			//  debug
			log.WithFields(log.Fields{
				"htmlLabel":	selector,
			}).Debug("parseHTMLStrImage success")

			return html
		}
	}

	// debug
	if len(html) <= 0 {
		//  debug
		log.WithFields(log.Fields{
			"htmlLabel":	selectors,
		}).Debug("html is empty by parseHTMLStrImage")
	}

	return html
}

// parseHTMLSet for get set meal html string and download image by parse doc, return html
func (s *SiteService) parseHTMLSet(doc *goquery.Document, pageURL string, imageDir string, selectors []string) [][]string {
	var dataList [][]string
	urlMD5 := ut.GetMD5(pageURL)

	for _, selector := range selectors {
		selection := iterativeHTML(doc.Selection, selector)  // get first
		if !checkSelectionLegal(selection, "html", selector, "parseHTMLStrImage") {
			continue
		}

		// get list
		selection.Each(func(i int, selc *goquery.Selection) {
			var data []string
			html, _ := selc.Html()
			if len(html) <= 0 {
				log.Debug("this selection of html is empty by parseHTMLImage")

				return
			}

			html = s.ReplaceImagePaths(html, pageURL, imageDir)
			data = append(data, urlMD5)
			data = append(data, html)
			dataList = append(dataList, data)
		})

		if len(dataList) > 0 {
			//  debug
			log.WithFields(log.Fields{
				"htmlLabel":	selector,
			}).Debug("parseHTMLStrImage success")

			break
		}
	}

	// debug
	if len(dataList) <= 0 {
		//  debug
		log.WithFields(log.Fields{
			"htmlLabel":	selectors,
		}).Debug("html is empty by parseHTMLStrImage")
	}

	return dataList
}


// parseHTMLSpec for get spec html string and download image by parse doc, return html
func (s *SiteService) parseHTMLSpec(doc *goquery.Document, pageURL string, imageDir string, selectors []string) *[][]string {
	var dataList *[][]string
	urlMD5 := ut.GetMD5(pageURL)

	for _, selector := range selectors {
		var levels []string
		dataList = iterativeLoopHTML(doc.Selection, selector, levels, 1, urlMD5, "", dataList)  // iterative till first list

		if len(*dataList) > 0 {
			//  debug
			log.WithFields(log.Fields{
				"htmlLabel":	selector,
			}).Debug("parseHTMLStrImage success")

			break
		}
	}

	// debug
	if len(*dataList) <= 0 {
		//  debug
		log.WithFields(log.Fields{
			"htmlLabel":	selectors,
		}).Debug("html is empty by parseHTMLStrImage")
	}

	return dataList
}

// parseJSONStrImage for get string and download image by parse doc, return html
func (s *SiteService) parseJSONStrImage(body []byte, pageURL string, imageDir string, selectors []string, listFlag bool) [][]string {
	var dataList [][]string
	urlMD5 := ut.GetMD5(pageURL)

	for _, selector := range selectors {
		labels := strings.Split(selector, cm.LabelSeparate)
		labelsNum := len(labels)
		jsonIter := jsoniter.Get(body, labels[0])
		for i := 1; i < labelsNum; i++ {  // cascade find label
			jsonIter = jsonIter.Get(labels[i])
		}

		if !checkSelectionLegal(jsonIter, "json", selector, "parseJSONStrImage") {
			continue
		}

		if listFlag {
			var data []string
			data = append(data, urlMD5)
			for i := 0; i < jsonIter.Size(); i++{
				str := jsonIter.Get(i).ToString()
				if len(str) <= 0 {
					log.Debug("this jsonIter is empty by parseJSONStrImage")

					continue
				}

				str = s.ReplaceImagePaths(str, pageURL, imageDir)
				data = append(data, str)
			}
			dataList = append(dataList, data)
		} else {
			str := jsonIter.ToString()
			if len(str) > 0 {
				str = s.ReplaceImagePaths(str, pageURL, imageDir)
				dataList = append(dataList, []string{str})
			}
		}


		if len(dataList) > 0 {
			//  debug
			log.WithFields(log.Fields{
				"jsonLabel":	selector,
			}).Debug("parseJSONStrImage success")

			break
		}
	}

	// debug
	if len(dataList) <= 0 {
		//  debug
		log.WithFields(log.Fields{
			"jsonLabel":	selectors,
		}).Debug("json is empty by parseJSONStrImage")
	}

	return dataList
}

// TaskDownload for download image task
func (s *SiteService) TaskDownload(data *sc.DataBlock) {
	name := data.Extra.(string)
	url := data.Message.(string)

	dir := cm.ImageDir + time.Now().Format("2006/01/02")
	if ok, _ := ut.PathExists(dir); !ok {
		err := os.MkdirAll(dir, os.ModePerm)
		if err != nil {
			log.WithFields(log.Fields{
				"url":		url,
				"dir":		dir,
				"error":	err.Error(),
			}).Error("can not make download dir by TaskDownload")

			return
		}
	}

	filePath := fmt.Sprintf("%s/%s", dir, name)
	if ok, _ := ut.PathExists(filePath); ok {
		return
	}

	// get response if not exist
	myResp := s.http.RequestTransportGet(url)
	if myResp == nil || len(myResp.Body) <= 0 {
		log.WithFields(log.Fields{
			"url":	url,
		}).Error("can not download this resource by TaskDownload")

		return
	}

	file, err := os.Create(filePath)
	if err != nil {
		log.WithFields(log.Fields{
			"url":		url,
			"filePath":	filePath,
			"error":	err.Error(),
		}).Error("can not create file by TaskDownload")

		return
	}

	defer file.Close()

	_, err = file.Write(myResp.Body)
	if err != nil {
		log.WithFields(log.Fields{
			"url":		url,
			"error":	err.Error(),
		}).Error("download image failed by TaskDownload")

		return
	}

	// only for debug
	log.WithFields(log.Fields{
		"imageURL":		url,
		"imageName":	name,
	}).Debug("download image success by TaskDownload")
}

// dispatch for dispatch task
func (s *SiteService) dispatch() {
	for {
		select {
			// do task of parse url
			case msg := <-s.downloadChan:
				//ctrl := &sc.ControlInfo{
				//	Name:    "http",  // must has value
				//	CtrlNum: 30,  // the size of concurrent routine pool
				//}
				data := &sc.DataBlock{
					Extra:   msg.Name,
					Message: msg.URL,
				}
				s.scheduler.AddTask(sc.Task{
					CtrlInfo:	nil,
					Data:   	data,
					DoTask: 	s.TaskDownload,
				})
		}
	}
}