/*
  Package sites for parse site template
*/

package sites

import (
	ut "siteResService/src/util"
	"time"

	"github.com/PuerkitoBio/goquery"
	log "github.com/sirupsen/logrus"

	cm "siteResService/src/common"
)

// ParseInfoCommonHTML for common html template which do request get returns pointer of ProInfo instance
func (s *SiteService) ParseInfoCommonHTML(pageURL string, doc *goquery.Document, orderDoc *goquery.Document, labels *cm.LabelsParse) *cm.ProInfo {
	var pi cm.ProInfo
	pi.PageURL = pageURL
	pi.Template = "templateCommonHTML"

	// get page html document
	if doc == nil {
		log.WithFields(log.Fields{
			"pageURL":	pageURL,
		}).Error("doc is nil, log by templateCommonHTML")

		return &pi
	}

	// cover image
	imageDir := cm.ImageDir + time.Now().Format("2006/01/02")
	pi.Cover = s.parseCoverImagesHTML(doc, pageURL, imageDir, labels.Cover)
	//pi.Images = images
	//if len(pi.Images) > 0 {
	//	pi.Cover = pi.Images[0].URL
	//}

	// title
	pi.Title = s.parseTitleHTML(doc, labels.Title)

	// price
	pi.Price = s.parsePriceHTML(doc, labels.Price)

	// description
	pi.Desc = ut.ToJson(s.parseDescHTML(doc, pageURL, imageDir, labels.Desc))

	if orderDoc == nil {  // main and order at the same page or can find order page
		// set meal
		pi.Set = s.parseGoodHTML(doc, pageURL, imageDir, labels.Good)

		// specifications
		pi.Spec = s.parseSpecHTML(doc, pageURL, imageDir, labels.Spec)
	} else {  // order at another page, request order page to get order document
		// set meal
		pi.Set = s.parseGoodHTML(orderDoc, pageURL, imageDir, labels.Good)

		// specifications
		pi.Spec = s.parseSpecHTML(orderDoc, pageURL, imageDir, labels.Spec)
	}

	return &pi
}