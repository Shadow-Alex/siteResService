/*
  Package sites for parse site template
*/

package sites

import (
	"encoding/csv"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	jsoniter "github.com/json-iterator/go"
	log "github.com/sirupsen/logrus"

	cm "siteResService/src/common"
	hs "siteResService/src/httpservice"
	sc "siteResService/src/scheduler"
	ut "siteResService/src/util"
)

type SiteService struct {
	http			*hs.ServiceHTTP
	scheduler		*sc.Scheduler
	SitesLabelMaps	sync.Map  // sites label maps
}

var instance *SiteService
var initTaskOnce sync.Once
var rootPath string
var goodKVMatch *regexp.Regexp
var goodValueMatch *regexp.Regexp

// GetSiteServiceInstance return siteService pointer instance
func GetSiteServiceInstance() *SiteService {
	initTaskOnce.Do(func() {
		instance = new(SiteService)
		instance.init()

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

	s.http = hs.GetHTTPInstance()
	s.scheduler = sc.GetScheduler()
	s.initSitesLabelMaps()

	// regexp for get style and value
	goodKVMatch = regexp.MustCompile(`\{(.*)\}`)
	goodValueMatch = regexp.MustCompile(`\((.*)\)`)
}

// note !!!
// each label in labels struct support find cascade html label (cascade json label) that to get accurate result
// for html, each cascade label use "|" to separate single labels,
// and html will use labels in "Order" to get order page href to request order page html, "|" usage for example:
// ".col-md-6|.title" means to find value in <div class="col-md-6"><div class="title">value</div></div>
// for web driver, the usage of separate "|" is the same as html,
// and use ";" to indicates that the previous layer is list, for example:
// "#data_foreach1|.compose_select;" means to get value list in data:
// <div id="data_foreach1"><div id="big0" class="compose_select">value</div><div id="big1" class="compose_select">value</div></div>
// and use "{title:value}" to indicates of the titles and values of specifications, for example:
// "{.rows-head:.rows-params};" means to get titles of specifications in label ".rows-head" and get values of specifications in label ".rows-params"
// <div class="rows-head">title1</div><div class="rows-params">value1</div>
// <div class="rows-head">title2</div><div class="rows-params">value2</div>
// and web driver will use labels in "Order" to do click and redirect to order page;
// for json, the usage of separate "|" is the same as html,
// and use ";" to indicates that the previous layer is list, ";" only use for cover, title, price, and only use once, for example:
// "data|products|covers;name" means to get value in data: {product:{covers:[name:value,name:value]}}
// addSiteResource for add site resource into SitesLabelMaps templates,
// the sequence of params []string : domain,character,order,cover,title,price,desc,spec,goods,pageURL,type
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
	goodLabels := strings.Split(record[7], "+")
	if len(record[7]) <= 0 || record[7] == "" {
		goodLabels = []string{}
	}
	specLabels := strings.Split(record[8], "+")
	if len(record[8]) <= 0 || record[8] == "" {
		specLabels = []string{}
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
		for _, good := range goodLabels {
			lab.Good = append(lab.Good, good)
		}
		for _, spec := range specLabels {
			lab.Spec = append(lab.Spec, spec)
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
		Good:		goodLabels,
		Spec:		specLabels,
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
	//	Good:		[]string{`.select-size`},
	//}
	// the sequence of params []string : domain,character,order,cover,title,price,desc,spec,goods,pageURL,type
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
	//	Good:		[]string{`.rows-id-params-select`},
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

// replaceImagePaths returns list of description images
func (s *SiteService) replaceImagePaths(desc string, pageURL string) []string {
	var images []string
	//resourceMap := make(map[string]string)
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
		images = append(images, imageURL)

		//resourceMap[imageSrc] = imageURL
		//imageName := generateImageName(imageURL, "")
		//downloadInfo := cm.DownLoadInfo{
		//	URL:	imageURL,
		//	Name:	imageName,
		//}
		//s.downloadChan <- downloadInfo  // push to download task, asynchronous download
		//imageName, okD := s.http.DownloadImage(imageURL, imageDir, "")
		//resourceMap[imageSrc] = imageDir + "/" + imageName
	})

	docDesc.Find("video").Each(func(i int, selection *goquery.Selection) {
		var videos []string
		videoSrc, ok := selection.Attr("src")
		if !ok && len(videoSrc) <= 0 {
			videoSrc, ok = selection.Attr("poster")
			if !ok && len(videoSrc) <= 0 {
				selection.Find("source").Each(func(j int, selection *goquery.Selection) {
					videoSrc, ok = selection.Attr("src")
					if !ok && len(videoSrc) <= 0 {
						return
					}
					videos = append(videos, videoSrc)
				})

				if len(videoSrc) <= 0 && len(videos) <= 0 {
					return
				}
			}
		}

		if len(videoSrc) > 0 {
			vURL := GetResourceURL(videoSrc, pageURL)
			images = append(images, vURL)
		}
		for _, video := range videos {
			vURL := GetResourceURL(video, pageURL)
			images = append(images, vURL)
		}


			//resourceMap[videoSrc] = vURL
			//imageName := generateImageName(vURL, "")
			//downloadInfo := cm.DownLoadInfo{
			//	URL:	vURL,
			//	Name:	imageName,
			//}
			//s.downloadChan <- downloadInfo  // push to download task, asynchronous download
			//imageName, okD := s.http.DownloadImage(vURL, imageDir, "")
			//resourceMap[videoSrc] = imageDir + "/" + imageName
		//}

		//posterSrc, okP := selection.Attr("poster")
		//if okP && len(posterSrc) > 0 {
		//	imgURL := GetResourceURL(posterSrc, pageURL)
		//	images = append(images, imgURL)

			//resourceMap[posterSrc] = imgURL
			//imageName := generateImageName(imgURL, "")
			//downloadInfo := cm.DownLoadInfo{
			//	URL:	imgURL,
			//	Name:	imageName,
			//}
			//s.downloadChan <- downloadInfo  // push to download task, asynchronous download
			//imageName, okD := s.http.DownloadImage(imgURL, imageDir, "")
			//resourceMap[posterSrc] = imageDir + "/" + imageName
		//}
		//
		//selection.Find("source").Each(func(j int, selection *goquery.Selection) {
		//	sourceSrc, ok := selection.Attr("src")
		//	if ok && len(sourceSrc) > 0 {
		//		vURL := GetResourceURL(sourceSrc, pageURL)
		//		images = append(images, vURL)

				//resourceMap[sourceSrc] = vURL
				//imageName := generateImageName(vURL, "")
				//downloadInfo := cm.DownLoadInfo{
				//	URL:	vURL,
				//	Name:	imageName,
				//}
				//s.downloadChan <- downloadInfo  // push to download task, asynchronous download
				//imageName, okD := s.http.DownloadImage(vURL, imageDir, "")
				//resourceMap[sourceSrc] = imageDir + "/" + imageName
		//	}
		//})
	})

	// replace image path to image url in html
	//for k, v := range resourceMap {
	//	v = strings.ReplaceAll(v, cm.ImageDir, cm.ImagePrefixDefault)
	//	desc = strings.ReplaceAll(desc, k, v)
	//}

	//logs.Info(desc)
	return images
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
func (s *SiteService) parseCoverImages(selection *goquery.Selection, pageURL string, imageDir string) []string {
	//var images []cm.ImageInfo
	var images []string
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
		images = append(images, imageURL)

		//imageName := generateImageName(imageURL, "")
		//downloadInfo := cm.DownLoadInfo{
		//	URL:	imageURL,
		//	Name:	imageName,
		//}
		//s.downloadChan <- downloadInfo  // push to download task, asynchronous download
		//imageName, ok := s.http.DownloadImage(imageURL, imageDir, "")
		//images = append(images, NewImage(imageURL, imageName, imageDir, true))
	})

	return images
}

// parseCoverImagesHTML parse by html, returns list of cover ImageInfo instance
func (s *SiteService) parseCoverImagesHTML(doc *goquery.Document, pageURL string, imageDir string, selectors []string) []string {
	//var images []cm.ImageInfo
	for _, selector := range selectors {
		labelList := strings.Split(selector, cm.ListSeparate)
		selection := iterativeHTML(doc.Selection, labelList[0])  // get first
		if !checkSelectionLegal(selection, "html", selector, "parseCoverImagesHTML") {
			continue
		}

		var images []string
		selection.Each(func(i int, selection1 *goquery.Selection) {
			selc := selection1
			if len(labelList) > 1 && len(labelList[1]) > 0 {
				selc = iterativeHTML(selection1, labelList[1]) // get first
				if !checkSelectionLegal(selc, "html", selector, "parseCoverImagesHTML") {
					return
				}
			}

			imgs := s.parseCoverImages(selc, pageURL, imageDir)
			for j := 0; j < len(imgs); j++ {
				images = append(images, imgs[j])
			}
		})

		if len(images) > 0 {
			//  debug
			log.WithFields(log.Fields{
				"selector":	selector,
			}).Debug("ParseCoverImagesHTML success")

			return images
		}
	}

	//  debug
	log.WithFields(log.Fields{
		"selectors":	selectors,
	}).Debug("can not get cover by ParseCoverImagesHTML")

	return []string{}
}

// iterativeJSON for iterative locate selection return located pointer of jsoniter.Any
func iterativeJSON(jIter jsoniter.Any, labels []string) jsoniter.Any {
	for i := 0; i < len(labels); i++ {  // cascade find label
		jIter = jIter.Get(labels[i])
		if !checkSelectionLegal(jIter, "json", strings.Join(labels, "|"), "iterativeJSON") {
			break
		}
	}

	return jIter
}

// parseCoverImagesJSON parse by json, returns list of cover ImageInfo instance
func (s *SiteService) parseCoverImagesJSON(body []byte, pageURL string, imageDir string, selectors []string) []string {
	//var images []cm.ImageInfo
	var images []string
	for _, selector := range selectors {
		labelList := strings.Split(selector, cm.ListSeparate)
		labels := strings.Split(labelList[0], cm.LabelSeparate)
		jIter := jsoniter.Get(body, labels[0])
		jIter = iterativeJSON(jIter, labels[1: ])
		if !checkSelectionLegal(jIter, "json", selector, "ParseCoverImagesJSON") {
			continue
		}

		var imageURL string
		if len(labelList) > 1 && len(labelList[1]) > 0 { // if contains ";", means labelList[1] indicates that the previous layer is list
			iterNum := jIter.Size()
			for i := 0; i < iterNum; i++ {
				iter := iterativeJSON(jIter.Get(i), strings.Split(labelList[1], cm.LabelSeparate))
				if !checkSelectionLegal(iter, "json", selector, "parseTitleJSON") {
					continue
				}

				imageURL = GetResourceURL(iter.ToString(), pageURL)
				images = append(images, imageURL)

				//imageName := generateImageName(imageURL, "")
				//downloadInfo := cm.DownLoadInfo{
				//	URL:	imageURL,
				//	Name:	imageName,
				//}
				//s.downloadChan <- downloadInfo  // push to download task, asynchronous download
				//imageName, ok := s.http.DownloadImage(imageURL, imageDir, "")
				//images = append(images, NewImage(imageURL, imageName, imageDir, true))
			}
		} else {  // if do not contains ";" means do not has list layer
			imageURL = GetResourceURL(jIter.ToString(), pageURL)
			images = append(images, imageURL)

			//imageName := generateImageName(imageURL, "")
			//downloadInfo := cm.DownLoadInfo{
			//	URL:	imageURL,
			//	Name:	imageName,
			//}
			//s.downloadChan <- downloadInfo  // push to download task, asynchronous download
			//imageName, ok := s.http.DownloadImage(imageURL, imageDir, "")
			//images = append(images, NewImage(imageURL, imageName, imageDir, true))
		}

		if len(images) > 0 {
			//  debug
			log.WithFields(log.Fields{
				"selector":	selector,
			}).Debug("ParseCoverImagesJSON success")

			return images
		}
	}

	//  debug
	log.WithFields(log.Fields{
		"selectors":	selectors,
	}).Debug("can not get cover by ParseCoverImagesJSON")

	return images
}

// parseTitleHTML parse by html, returns title after parse
func (s *SiteService) parseTitleHTML(doc *goquery.Document, selectors []string) string {
	for _, selector := range selectors {
		labelList := strings.Split(selector, cm.ListSeparate)
		selection := iterativeHTML(doc.Selection, labelList[0])  // get first
		if !checkSelectionLegal(selection, "html", selector, "parseTitleHTML") {
			continue
		}

		var title string
		var titles []string
		selection.Each(func(i int, selection1 *goquery.Selection) {
			selc := selection1
			if len(labelList) > 1 && len(labelList[1]) > 0 {
				selc = iterativeHTML(selection1, labelList[1]) // get first
				if !checkSelectionLegal(selc, "html", selector, "parseTitleHTML") {
					return
				}
			}

			text := strings.TrimSpace(selc.Text())
			if len(text) > 0 {
				titles = append(titles, text)
			}
		})
		title = strings.Join(titles, ",")  // "," for separate each title

		if len(title) > 0 {
			//  debug
			log.WithFields(log.Fields{
				"selector":	selector,
			}).Debug("parseTitleHTML success")

			return title
		}
	}

	//  debug
	log.WithFields(log.Fields{
		"selectors":	selectors,
	}).Debug("can not get title by ParseTitleHTML")

	return ""
}

// parseTitleJSON parse by json, returns title after parse
func (s *SiteService) parseTitleJSON(body []byte, selectors []string) string {
	for _, selector := range selectors {
		labelList := strings.Split(selector, cm.ListSeparate)
		labels := strings.Split(labelList[0], cm.LabelSeparate)
		jIter := jsoniter.Get(body, labels[0])
		jIter = iterativeJSON(jIter, labels[1: ])
		if !checkSelectionLegal(jIter, "json", selector, "ParseTitleJSON") {
			continue
		}

		var title string  // final title
		var titles []string  // if contains multi title
		if len(labelList) > 1 && len(labelList[1]) > 0 {  // if contains ";", means labelList[1] indicates that the previous layer is list
			iterNum := jIter.Size()
			for i := 0; i < iterNum; i++ {
				iter := iterativeJSON(jIter.Get(i), strings.Split(labelList[1], cm.LabelSeparate))
				if !checkSelectionLegal(iter, "json", selector, "parseTitleJSON") {
					continue
				}

				titles = append(titles, strings.TrimSpace(iter.ToString()))
			}
			title = strings.Join(titles, ",")  // "," for separate each title
		} else {
			title = strings.TrimSpace(jIter.ToString())
		}

		if len(title) > 0 {
			//  debug
			log.WithFields(log.Fields{
				"selector":	selector,
			}).Debug("ParseTitleJSON success")

			return title
		}
	}

	//  debug
	log.WithFields(log.Fields{
		"selectors":	selectors,
	}).Debug("can not get title by ParseTitleJSON")

	return ""
}

// parsePriceHTML parse by html, returns price after parse
func (s *SiteService) parsePriceHTML(doc *goquery.Document, selectors []string) string {
	for _, selector := range selectors {
		labelList := strings.Split(selector, cm.ListSeparate)
		selection := iterativeHTML(doc.Selection, labelList[0])  // get first
		if !checkSelectionLegal(selection, "html", selector, "parsePriceHTML") {
			continue
		}

		var price string
		var prices []string
		selection.Each(func(i int, selection1 *goquery.Selection) {
			selc := selection1
			if len(labelList) > 1 && len(labelList[1]) > 0 {
				selc = iterativeHTML(selection1, labelList[1]) // get first
				if !checkSelectionLegal(selc, "html", selector, "parsePriceHTML") {
					return
				}
			}

			text := strings.TrimSpace(selc.Text())
			if len(text) > 0 {
				prices = append(prices, text)
			}
		})
		price = strings.Join(prices, ",")  // "," for separate each title

		if len(price) > 0 {
			//  debug
			log.WithFields(log.Fields{
				"selector":	selector,
			}).Debug("ParsePriceHTML success")

			return price
		}
	}

	//  debug
	log.WithFields(log.Fields{
		"selectors":	selectors,
	}).Debug("can not get price by ParsePriceHTML")

	return ""
}

// parsePriceJSON parse by json, returns price after parse
func (s *SiteService) parsePriceJSON(body []byte, selectors []string) string {
	for _, selector := range selectors {
		labelList := strings.Split(selector, cm.ListSeparate)
		labels := strings.Split(labelList[0], cm.LabelSeparate)
		jIter := jsoniter.Get(body, labels[0])
		jIter = iterativeJSON(jIter, labels[1: ])
		if !checkSelectionLegal(jIter, "json", selector, "ParsePriceJSON") {
			continue
		}

		var priceHTMLs []string  // if contains multi price
		if len(labelList) > 1 && len(labelList[1]) > 0 {  // if contains ";", means labelList[1] indicates that the previous layer is list
			iterNum := jIter.Size()
			for i := 0; i < iterNum; i++ {
				iter := iterativeJSON(jIter.Get(i), strings.Split(labelList[1], cm.LabelSeparate))
				if !checkSelectionLegal(iter, "json", selector, "ParsePriceJSON") {
					continue
				}

				priceHTMLs = append(priceHTMLs, iter.ToString())
			}
		} else {
			priceHTMLs = append(priceHTMLs, jIter.ToString())
		}

		var prices []string  // final price
		for i := 0; i < len(priceHTMLs); i++ {
			doc, err := goquery.NewDocumentFromReader(strings.NewReader(priceHTMLs[i]))
			if err != nil {
				log.WithFields(log.Fields{
					"html":	priceHTMLs[i],
				}).Debug("can not parse string to html by ParsePriceJSON")

				continue
			}

			text := strings.TrimSpace(doc.Text())
			if len(text) > 0 {
				prices = append(prices, text)
			}
		}

		price := strings.Join(prices, ",")  // "," for separate each title
		if len(price) > 0 {
			//  debug
			log.WithFields(log.Fields{
				"selector":	selector,
			}).Debug("ParsePriceJSON success")

			return price
		}
	}

	//  debug
	log.WithFields(log.Fields{
		"selectors":	selectors,
	}).Debug("can not get price by ParsePriceJSON")

	return ""
}

// iterativeHTML for iterative locate selection return located selection
func iterativeHTML(selection *goquery.Selection, selector string) *goquery.Selection {
	labels := strings.Split(selector, cm.LabelSeparate)
	for i := 0; i < len(labels); i++ {  // cascade find label
		selection = selection.Find(labels[i])
		if !checkSelectionLegal(selection, "html", selector, "iterativeHTML") {
			break
		}
	}

	return selection
}

// splitDescHTMLParse for split description html and parse by order, return desc info list
func (s *SiteService) splitDescHTMLParse(html string, pageURL string) []string {
	var descInfo []string

	htmlList := strings.Split(html, "\n")
	for _, h := range htmlList {
		h = strings.TrimSpace(h)  // remove space from head and tail
		if len(h) <= 0 {
			continue
		}

		doc, err := goquery.NewDocumentFromReader(strings.NewReader(h))
		if err != nil {
			log.WithFields(log.Fields{
				"html":	h,
			}).Debug("can not parse string to html by splitDescParseHTML")

			continue
		}

		if len(doc.Text()) > 0 {  // get text description
			descInfo = append(descInfo, strings.TrimSpace(doc.Text()))
		}

		images := s.replaceImagePaths(h, pageURL)  // get image description
		if len(images) <= 0 {
			continue
		}

		for _, image := range images {
			descInfo = append(descInfo, image)
		}
	}

	return descInfo
}

// parseDescHTML for get desc html string and download image by parse doc, return desc info list
func (s *SiteService) parseDescHTML(doc *goquery.Document, pageURL string, imageDir string, selectors []string) []string {
	for _, selector := range selectors {
		labelList := strings.Split(selector, cm.ListSeparate)
		selection := iterativeHTML(doc.Selection, labelList[0])  // get first
		if !checkSelectionLegal(selection, "html", selector, "parsePriceHTML") {
			continue
		}

		var dataInfos []string
		selection.Each(func(i int, selection1 *goquery.Selection) {
			selc := selection1
			if len(labelList) > 1 && len(labelList[1]) > 0 {
				selc = iterativeHTML(selection1, labelList[1]) // get first
				if !checkSelectionLegal(selc, "html", selector, "parsePriceHTML") {
					return
				}
			}

			html, _ := selc.Html()
			if len(html) > 0 {
				descs := s.splitDescHTMLParse(html, pageURL)
				for i := 0; i < len(descs); i++ {
					dataInfos = append(dataInfos, descs[i])
				}
			}
		})

		if len(dataInfos) > 0 {
			//  debug
			log.WithFields(log.Fields{
				"selector":	selector,
			}).Debug("ParsePriceHTML success")

			return dataInfos
		}
	}

	// debug
	log.WithFields(log.Fields{
		"selectors":	selectors,
	}).Debug("can not get desc by parseDescHTML")

	return []string{}
}

// parseDescJSON for get string and download image by parse doc, return desc info list
func (s *SiteService) parseDescJSON(body []byte, pageURL string, selectors []string) []string {
	for _, selector := range selectors {
		labelList := strings.Split(selector, cm.ListSeparate)
		labels := strings.Split(labelList[0], cm.LabelSeparate)
		jIter := jsoniter.Get(body, labels[0])
		jIter = iterativeJSON(jIter, labels[1: ])
		if !checkSelectionLegal(jIter, "json", selector, "parseDescJSON") {
			continue
		}

		var descHTMLs []string  // if contains multi price
		if len(labelList) > 1 && len(labelList[1]) > 0 {  // if contains ";", means labelList[1] indicates that the previous layer is list
			iterNum := jIter.Size()
			for i := 0; i < iterNum; i++ {
				iter := iterativeJSON(jIter.Get(i), strings.Split(labelList[1], cm.LabelSeparate))
				if !checkSelectionLegal(iter, "json", selector, "parseDescJSON") {
					continue
				}

				descHTMLs = append(descHTMLs, iter.ToString())
			}
		} else {
			descHTMLs = append(descHTMLs, jIter.ToString())
		}

		var dataInfos []string  // final price
		for i := 0; i < len(descHTMLs); i++ {
			descs := s.splitDescHTMLParse(descHTMLs[i], pageURL)
			for j := 0; j < len(descs); j++ {
				dataInfos = append(dataInfos, descs[j])
			}
		}

		if len(dataInfos) > 0 {
			//  debug
			log.WithFields(log.Fields{
				"selector":	selector,
			}).Debug("parseDescJSON success")

			return dataInfos
		}
	}

	//  debug
	log.WithFields(log.Fields{
		"selectors":	selectors,
	}).Debug("can not get desc by parseDescJSON")

	return []string{}
}

// parseGoodHTML for get goods html string and download image by parse doc, return goods info list
func (s *SiteService) parseGoodHTML(doc *goquery.Document, pageURL string, imageDir string, selectors []string) [][]string {
	urlMD5 := ut.GetMD5(pageURL)
	for _, selector := range selectors {
		labelList := strings.Split(selector, cm.ListSeparate)
		selection := iterativeHTML(doc.Selection, labelList[0])  // get first
		if !checkSelectionLegal(selection, "html", selector, "parseGoodHTML") {
			continue
		}

		var dataInfos [][]string
		// get list
		selection.Each(func(i int, selection1 *goquery.Selection) {
			selc := selection1
			if len(labelList) > 1 && len(labelList[1]) > 0 {
				selc = iterativeHTML(selection1, labelList[1]) // get first
				if !checkSelectionLegal(selc, "html", selector, "parseGoodHTML") {
					return
				}
			}

			html, _ := selc.Html()
			if len(html) <= 0 {
				log.Debug("this selection of html is empty by parseGoodHTML")

				return
			}

			text := selc.Text()
			images := s.replaceImagePaths(html, pageURL)  // get image description
			if len(images) <= 0 {  // there is no images
				var data []string
				data = append(data, urlMD5)
				data = append(data, text)
				data = append(data, "")
				dataInfos = append(dataInfos, data)
			}
			for _, image := range images {
				var data []string
				data = append(data, urlMD5)
				data = append(data, text)
				data = append(data, image)
				dataInfos = append(dataInfos, data)
			}
		})

		if len(dataInfos) > 0 {
			//  debug
			log.WithFields(log.Fields{
				"selector":	selector,
			}).Debug("parseGoodHTML success")

			return dataInfos
		}
	}

	//  debug
	log.WithFields(log.Fields{
		"selectors":	selectors,
	}).Debug("can not get goods by parseGoodHTML")

	return [][]string{}
}



// parseGoodJSON for get string and download image by parse doc, return goods info list
func (s *SiteService) parseGoodJSON(body []byte, pageURL string, selectors []string) [][]string {
	urlMD5 := ut.GetMD5(pageURL)
	for _, selector := range selectors {
		labelList := strings.Split(selector, cm.ListSeparate)
		labels := strings.Split(labelList[0], cm.LabelSeparate)
		jIter := jsoniter.Get(body, labels[0])
		jIter = iterativeJSON(jIter, labels[1: ])
		if !checkSelectionLegal(jIter, "json", selector, "parseGoodJSON") {
			continue
		}

		var dataInfos [][]string
		if len(labelList) > 1 && len(labelList[1]) > 0 {  // if contains ";", means labelList[1] indicates that the previous layer is list
			iterNum := jIter.Size()
			for i := 0; i < iterNum; i++ {
				iter := iterativeJSON(jIter.Get(i), strings.Split(labelList[1], cm.LabelSeparate))
				if !checkSelectionLegal(iter, "json", selector, "parseGoodJSON") {
					continue
				}

				text := iter.ToString()
				if len(text) > 0 {
					var data []string
					data = append(data, urlMD5)
					data = append(data, text)  // add good text
					data = append(data, "")  // add good image
					dataInfos = append(dataInfos, data)
				}
			}
		} else {
			text := jIter.ToString()
			if len(text) > 0 {
				var data []string
				data = append(data, urlMD5)
				data = append(data, text)  // add good data
				data = append(data, "")  // add good image
				dataInfos = append(dataInfos, data)
			}
		}

		if len(dataInfos) > 0 {
			//  debug
			log.WithFields(log.Fields{
				"selector":	selector,
			}).Debug("parseGoodJSON success")

			return dataInfos
		}
	}

	//  debug
	log.WithFields(log.Fields{
		"selectors":	selectors,
	}).Debug("can not get goods by parseGoodJSON")

	return [][]string{}
}

// TODO: can not get mapping to goods
// iterativeLoopHTML return pointer list of spec data
// levels for match spec data and goods data, first goods data match level1 spec data
func iterativeLoopHTML(selection *goquery.Selection, selector string, urlMD5 string, style string, levels []string, level int, dataList *[][]string) *[][]string {
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
	var styleTitles []string
	iterSelector := selector[: index]
	// match style and its value's labels
	subG := goodKVMatch.FindStringSubmatch(iterSelector)
	if len(subG) > 1 {  // sub[0] is the origin str, ignore it, only use sub[1]
		values := strings.Split(subG[1], ":")
		styleSection := selection  // separate from get style value and style title
		styleSection.Find(values[0]).Each(func(i int, selc *goquery.Selection) { // get style title
			styleTitles = append(styleTitles, strings.TrimSpace(selc.Text()))
		})
		iterSelector = values[1]  // get style value's labels
	}

	// travers all the list label marked by ";"
	selection.Each(func(i int, selc *goquery.Selection) {
		var style string
		// pass style title params only if the num of style titles match the num of style style values
		if len(styleTitles) > 0 && len(styleTitles) == selection.Size(){
			style = styleTitles[i]
		} else {
			levels = append(levels, "level" + strconv.Itoa(level) + "-" + strconv.Itoa(i))
		}

		dataList = iterativeLoopHTML(selc, selector[index + 1 :], urlMD5, style, levels, level + 1, dataList) // process rest, and +1 to avoid cm.ListSeparate
	})

	return dataList
}

// parseSpecHTML for get spec html string and download image by parse doc, return spec info list
func (s *SiteService) parseSpecHTML(doc *goquery.Document, pageURL string, imageDir string, selectors []string) [][]string {
	urlMD5 := ut.GetMD5(pageURL)
	for _, selector := range selectors {
		//var levels []string
		//dataList = *iterativeLoopHTML(doc.Selection, selector, urlMD5, "", levels, 1, &dataList)  // iterative till first list
		labelList := strings.Split(selector, cm.ListSeparate)
		selection := iterativeHTML(doc.Selection, labelList[0])  // get first
		if !checkSelectionLegal(selection, "html", selector, "parseSpecHTML") {
			continue
		}

		// get style and its labels
		var dataInfos [][]string
		var styleTitles []string
		var valueSelector string
		if len(labelList) > 1 && len(labelList[1]) > 0 {
			sub := goodKVMatch.FindStringSubmatch(labelList[1]) // match style labels
			if len(sub) > 1 {
				values := strings.Split(sub[1], ":")
				styleSection := selection // separate from get style value and style title
				styleMap := make(map[string]int)
				styleSection.Find(values[0]).Each(func(i int, selc *goquery.Selection) { // get style title
					sText := strings.TrimSpace(selc.Text())
					_, ok := styleMap[sText]  // avoid duplicate style
					if !ok {
						styleMap[sText] = 1
						styleTitles = append(styleTitles, sText)
					}
				})
				valueSelector = values[1] // get style value's labels
			}
		}

		bigNum := len(styleTitles)
		bigNumFloat := float64(bigNum)
		selection.Each(func(i int, selection1 *goquery.Selection) {
			selection1.Find(valueSelector).Each(func(j int, selection2 *goquery.Selection) {
				selection2.Children().Each(func(k int, selection3 *goquery.Selection) {
					html, err := selection3.Html()
					if err != nil {
						return
					}

					style := styleTitles[j % bigNum]  // use Module bigNum division to get corresponding style
					bigID := strconv.Itoa(int(math.Floor(float64(j) / bigNumFloat)))  // use floor to get big id
					text := selection3.Text()
					images := s.replaceImagePaths(html, pageURL)  // get image description
					if len(images) <= 0 {  // there is no images
						var data []string
						data = append(data, urlMD5)
						data = append(data, style)
						data = append(data, "good_" + strconv.Itoa(i) + "-num_" + bigID)
						data = append(data, strings.TrimSpace(text))
						data = append(data, "")
						dataInfos = append(dataInfos, data)
					}
					for _, image := range images {
						var data []string
						data = append(data, urlMD5)
						data = append(data, style)
						data = append(data, "good_" + strconv.Itoa(i) + "-num_" + bigID)
						data = append(data, strings.TrimSpace(text))
						data = append(data, image)
						dataInfos = append(dataInfos, data)
					}
				})
			})
		})

		if len(dataInfos) > 0 {
			//  debug
			log.WithFields(log.Fields{
				"selector":	selector,
			}).Debug("parseSpecHTML success")

			return dataInfos
		}
	}

	//  debug
	log.WithFields(log.Fields{
		"selectors":	selectors,
	}).Debug("can not get spec by parseSpecHTML")

	return [][]string{}
}

// TODO: can not get mapping to goods
// iterativeLoopJSON return pointer list of spec data
// levels for match spec data and goods data, first goods data match level1 spec data
func iterativeLoopJSON(jIter jsoniter.Any, selector string, urlMD5 string, styleKey string,
	goodID int, numID int, dataList *[][]string) *[][]string {
	index := strings.Index(selector, cm.ListSeparate)
	if index == -1 {
		// match style value's labels
		var styleValues []string
		subGV := goodValueMatch.FindStringSubmatch(selector)
		if len(subGV) > 1 {
			styleValues = strings.Split(subGV[1], ",")
		}

		var data []string
		data = append(data, urlMD5)
		data = append(data, styleKey)
		data = append(data, "good_" + strconv.Itoa(goodID) + "-num_" + strconv.Itoa(numID))
		jIterBak := jIter
		for i := 0; i < len(styleValues); i++ {  // add style values
			jIter = jIterBak
			jIter = iterativeJSON(jIter, strings.Split(styleValues[i], cm.LabelSeparate))
			data = append(data, jIter.ToString())
		}
		*dataList = append(*dataList, data)

		return dataList
	}

	// get mapping identification
	mappingFlag := false
	iterSelector := selector[: index]
	if strings.HasSuffix(iterSelector, "^") {
		iterSelector = strings.ReplaceAll(iterSelector, "^", "")  // remove mapping identification
		mappingFlag = true
	}

	// get style and its labels
	// match style and its value's labels
	subG := goodKVMatch.FindStringSubmatch(iterSelector)
	if len(subG) > 1 {  // subG[0] is the origin str, ignore it, only use subG[1]
		values := strings.Split(subG[1], ":")
		if len(values) > 1 {
			styleJIter := jIter  // separate from get style value and style title
			iter := iterativeJSON(styleJIter, strings.Split(values[0], cm.LabelSeparate))
			if checkSelectionLegal(iter, "json", values[0], "iterativeLoopJSON") {
				styleKey = iter.ToString()
			}
			iterSelector = values[1]  // get style value's labels
		}
	}

	iter := iterativeJSON(jIter, strings.Split(iterSelector, cm.LabelSeparate))
	if !checkSelectionLegal(iter, "json", iterSelector, "iterativeLoopJSON") {
		return dataList
	}

	// do ergodic
	size := iter.Size()
	if size > 0 {
		numIDBak := numID
		for i := 0; i < size; i++ {
			// recursive call
			if mappingFlag {
				goodID = i
			}
			if !mappingFlag && goodID >= 0 && numIDBak == -1 {  // goodID and numID are not located at same level
				numID = i
			}
			dataList = iterativeLoopJSON(iter.Get(i), selector[index + 1 :], urlMD5, styleKey, goodID, numID, dataList)
		}
	}

	return dataList
}

// TODO: can not get spec title
// parseSpecJSON for get string and download image by parse doc, return html
func (s *SiteService) parseSpecJSON(body []byte, pageURL string, selectors []string) [][]string {
	urlMD5 := ut.GetMD5(pageURL)
	for _, selector := range selectors {
		index := strings.Index(selector, cm.ListSeparate)
		if index == -1 {
			continue
		}

		var dataInfos [][]string
		jIter := jsoniter.Get(body)
		iterativeLoopJSON(jIter, selector, urlMD5, "", -1, -1, &dataInfos)

		if len(dataInfos) > 0 {
			//  debug
			log.WithFields(log.Fields{
				"selector":	selector,
			}).Debug("parseSpecJSON success")

			return dataInfos
		}
	}

	//  debug
	log.WithFields(log.Fields{
		"selectors":	selectors,
	}).Debug("can not get spec by parseSpecJSON")

	return [][]string{}
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