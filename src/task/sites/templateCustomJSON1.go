/*
  Package sites for parse site template
*/

package sites

import (
	"regexp"
	"time"

	jsoniter "github.com/json-iterator/go"
	log "github.com/sirupsen/logrus"

	cm "../../common"
	hs "../../httpService"
)

/**
https://mall.colapatheboutique.com/my/t1/1724
*/
// ParseInfoCustomJSON1 for GET JSON template returns pointer of ProInfo instance
func (s *SiteService) ParseInfoCustomJSON1(pageURL string) *cm.ProInfo {
	var pi cm.ProInfo
	pi.PageURL = pageURL
	pi.Template = "templateCustomJson1"
	
	T1Pattern, _ := regexp.Compile("t1/(\\d+)")
	ids := T1Pattern.FindStringSubmatch(pageURL)
	if len(ids) <= 0 {
		log.WithFields(log.Fields{
			"pageURL":	pageURL,
		}).Error("find string sub match failed by templateCustomJson1")

		return &pi
	}

	dataUrl := cm.T1ServerURL + ids[1]
	headers := hs.DefaultHeader()
	headers["Accept-Language"] = "zh-CN,zh;q=0.9,en;q=0.8,zh-TW;q=0.7"
	headers["Accept"] = "application/json, text/plain, */*"
	resp := s.http.RequestGet(dataUrl, headers)
	if resp == nil || resp.StatusCode != 200 {
		log.WithFields(log.Fields{
			"pageURL":	pageURL,
		}).Error("http request get page failed by templateCustomJson1")

		return &pi
	}

	// head image
	var images []cm.ImageInfo
	// download image
	cover := jsoniter.Get(resp.Body, "info", "cover").ToString()
	imageSrc := cm.T1CND + cover
	imageURL := GetResourceURL(imageSrc, pageURL)
	imageDir := cm.ImageDir + time.Now().Format("2006/01/02")
	imageName, ok := s.http.DownloadImage(imageURL, imageDir, "")
	if ok {
		images = append(images, NewImage(imageURL, imageName, imageDir, true))
	}

	//logs.Info(covers)
	pi.Cover = images
	//if len(pi.Images) > 0 {
	//	pi.Cover = pi.Images[0].URL
	//}

	// title
	pi.Title = jsoniter.Get(resp.Body, "info", "name").ToString()

	// price
	pi.Price = jsoniter.Get(resp.Body, "info", "cover").ToString()

	//
	//pi.ID = ids[1] // doc.Find(".pw-s").Text()

	// description
	pi.Desc = jsoniter.Get(resp.Body, "info", "content").ToString()
	pi.Desc = s.ReplaceImagePaths(pi.Desc, pageURL, imageDir)

	// specifications
	//pi.Spec, _ = ""

	return &pi
}
