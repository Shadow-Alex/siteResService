/*
  Package sites for parse site template
*/

package sites

import (
	ut "siteResService/src/util"
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
	pi.Cover = s.parseCoverImagesJSON(body, pageURL, imageDir, labels.Cover)

	//logs.Info(covers)
	//pi.Cover = images
	//if len(pi.Images) > 0 {
	//	pi.Cover = pi.Images[0].URL
	//}

	// title
	pi.Title = s.parseTitleJSON(body, labels.Title)

	// price
	pi.Price = s.parsePriceJSON(body, labels.Price)

	// description
	pi.Desc = ut.ToJson(s.parseDescJSON(body, pageURL, labels.Desc))

	// specifications
	pi.Spec = s.parseSpecJSON(body, pageURL, labels.Spec)

	// set meal
	pi.Good = s.parseGoodJSON(body, pageURL, labels.Good)

	if len(pi.Good) > 0 {
		pi.Currency = s.parseCurrency(pi.Good[0][2])
	}

	return &pi
}
