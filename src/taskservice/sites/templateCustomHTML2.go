/*
  Package sites for parse site template
*/

package sites

import (
	"strings"

	"github.com/PuerkitoBio/goquery"
	log "github.com/sirupsen/logrus"

	cm "siteResService/src/common"
	hs "siteResService/src/httpservice"
)

// ParseInfoCustomHTML2 for custom html template returns pointer of ProInfo instance
func (s *SiteService) ParseInfoCustomHTML2(pageURL string) *cm.ProInfo {
	if strings.Contains(pageURL, "collections") {
		return s.parse2(pageURL)
	}

	return s.parse1(pageURL)
}

// https://twmuch.com/collections/air-fresher/
func (s *SiteService) parse2(pageURL string) *cm.ProInfo {
	log.Info("collections....parse")

	var pi cm.ProInfo
	pi.Template = "templateCustomHTML2"
	if strings.LastIndex(pageURL, "/") < len(pageURL)-1 {
		pi.PageURL = pageURL + "/ordervar.js"
	} else {
		pi.PageURL = pageURL + "ordervar.js"
	}

	resp := s.http.RequestGet(pi.PageURL, hs.DefaultHeader())
	if resp == nil || resp.StatusCode != 200 {
		log.WithFields(log.Fields{
			"pageURL":	pageURL,
		}).Error("http request get page failed by templateCustomHTML2")

		return &pi
	}

	return &pi
}

// parse1 for parse
func (s *SiteService) parse1(pageURL string) *cm.ProInfo {
	var pi cm.ProInfo
	pi.PageURL = pageURL
	pi.Template = "templateCustomHTML2"

	doc := s.http.GetDocRequestGet(pageURL)
	if doc == nil {
		log.WithFields(log.Fields{
			"pageURL":	pageURL,
		}).Error("http request get page failed by templateCustomHTML2")

		return &pi
	}

	var images []string
	coverPath, ok := doc.Find(".product_info>img").Attr("src")
	//imageDir := cm.ImageDir + time.Now().Format("2006/01/02")
	if ok {
		imageURL := GetResourceURL(coverPath, pageURL)
		images = append(images, imageURL)

		//imageName, ok := s.http.DownloadImage(imageURL, imageDir, "")
		//if ok {
		//	images = append(images, NewImage(imageURL, imageName, imageDir, true))
		//}
		pi.Cover = images
	}

	//if len(pi.Images) > 0 {
	//	pi.Cover = pi.Images[0].URL
	//}

	title := ""
	doc.Find(".title>h1").Each(func(i int, selection *goquery.Selection) {
		title += selection.Text() + " "
	})
	// title
	pi.Title = title

	// price
	pi.Price = doc.Find(".price>ins").Text()

	// description list - class=product_info second
	doc.Find(".product_info").Each(func(i int, selection *goquery.Selection) {
		if i == 1 {
			pi.Desc, _ = selection.Html()
		}
	})
	//pi.Desc = s.ReplaceImagePaths(pi.Desc, pageURL)

	return &pi
}
