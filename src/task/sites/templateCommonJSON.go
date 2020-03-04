/*
  Package sites for parse site template
*/

package sites

import (
	"time"

	log "github.com/sirupsen/logrus"

	cm "siteResService/src/common"
)

/**
https://www.sukiemall.com/catalog/household-merchandises/p/yjqjs
*/
// ParseInfoCOMMONJSON for common json template which do request post returns pointer of ProInfo instance
func (s *SiteService) ParseInfoCommonJSON(pageURL string, body []byte, labels *cm.LabelsParse) *cm.ProInfo {
	var pi cm.ProInfo
	pi.PageURL = pageURL
	pi.Template = "templateCommonJson"

	if body == nil || len(body) <= 0 {
		log.WithFields(log.Fields{
			"pageURL":	pageURL,
		}).Error("json body is empty, log by ParseInfoCommonJSON")

		return &pi
	}


	// head image
	imageDir := cm.ImageDir + time.Now().Format("2006/01/02")
	pi.Cover = s.ParseCoverImagesJSON(body, pageURL, imageDir, labels.Cover)

	//logs.Info(covers)
	//pi.Cover = images
	//if len(pi.Images) > 0 {
	//	pi.Cover = pi.Images[0].URL
	//}

	// title
	pi.Title = s.ParseTitleJSON(body, labels.Title)

	// price
	pi.Price = s.ParsePriceJSON(body, labels.Price)

	// description
	pi.Desc = s.parseDescJSON(body, labels.Desc)

	// set meal
	pi.Set = s.parseSetJSON(body, pageURL, labels.Set)

	// specifications
	pi.Spec = s.parseSpecJSON(body, pageURL, labels.Spec)

	return &pi
}
