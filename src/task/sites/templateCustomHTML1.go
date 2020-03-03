/*
  Package sites for parse site template
*/

package sites

import (
	"time"

	"github.com/PuerkitoBio/goquery"
	log "github.com/sirupsen/logrus"

	cm "../../common"
)

/**
https://www.ikigo.com.tw/index.php?route=product/product&path=4_255_371&product_id=6704600248
*/
// ParseInfoCustomHTML1 for custom html template returns pointer of ProInfo instance
func (s *SiteService) ParseInfoCustomHTML1(pageURL string) *cm.ProInfo {
	var pi cm.ProInfo
	pi.PageURL = pageURL
	pi.Template = "templateCustomHTML1"

	// get page html document
	doc := s.http.GetDocRequestGet(pageURL)
	if doc == nil {
		log.WithFields(log.Fields{
			"pageURL":	pageURL,
		}).Error("http request get page failed by templateCustomHTML1")

		return &pi
	}

	// head image
	var images []cm.ImageInfo
	imageDir := cm.ImageDir + time.Now().Format("2006/01/02")
	doc.Find("#newpage>a").Each(func(i int, selection *goquery.Selection) {
		imageSrc, ok := selection.Attr("href")
		if !ok && len(imageSrc) <= 0 {
			return
		}

		// download image
		imageURL := GetResourceURL(imageSrc, pageURL)
		imageName, ok := s.http.DownloadImage(imageURL, imageDir, "")
		if ok {
			images = append(images, NewImage(imageURL, imageName, imageDir, true))
		}
	})

	pi.Cover = images
	//if len(pi.Images) > 0 {
	//	pi.Cover = pi.Images[0].URL
	//}

	// title
	pi.Title = doc.Find(".pw-h").Text()

	// price
	pi.Price = doc.Find("#price_span").Text()

	// description
	pi.Desc, _ = doc.Find("#tab1").Html()
	pi.Desc = s.ReplaceImagePaths(pi.Desc, pageURL, imageDir)

	// specifications
	//pi.Spec, _ = doc.Find("#tab3").Html()
	//pi.Spec = s.ReplaceImagePaths(pi.Spec, pageURL, imageDir)

	return &pi
}