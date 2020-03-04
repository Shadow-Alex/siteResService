/*
  Package http for request
*/

package http

import (
	"os"
	"fmt"
	"time"
	"bytes"
	"strings"
	"sync"
	"sync/atomic"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"

	"github.com/PuerkitoBio/goquery"
	"github.com/tebeka/selenium"
	"github.com/tebeka/selenium/firefox"
	"github.com/astaxie/beego"
	log "github.com/sirupsen/logrus"

	cm "siteResService/src/common"
	ut "siteResService/src/util"
)

// ServiceHTTP for http request
type ServiceHTTP struct {
	wd					*selenium.WebDriver
	wdLastRestartTime	time.Time  // for restart web driver
	RequestCounter		uint64  // calculation num of http request, must use by atomic !!!
}

var instance *ServiceHTTP
var initHTTPOnce sync.Once

type CustomResponse struct {
	StatusCode int
	Body       []byte
	Headers    map[string]string
}

// GetHTTPInstance return ServiceHTTP pointer if create http service success
func GetHTTPInstance() *ServiceHTTP {
	initHTTPOnce.Do(func() {
		instance = new(ServiceHTTP)
		instance.init()

		log.Info("init http service instance success...")
	})

	return instance
}

// init ServiceHTTP client
func (h *ServiceHTTP) init() {
	h.wd = initWebDriver()
	h.wdLastRestartTime = time.Now()

	atomic.StoreUint64(&h.RequestCounter, 0) // init counter to 0
}

// DefaultHeader for default header
func DefaultHeader() map[string]string {
	headers := make(map[string]string)
	headers["User-Agent"] = cm.HeaderUserAgent
	return headers
}

// getTransportHttpClient for get http client with transport
func getTransportHttpClient() *http.Client {
	timeout := beego.AppConfig.DefaultInt("http::timeout", cm.HTTPTimeOut)

	connHTTP := &http.Client{
		Transport: &http.Transport{
			Dial: func(netw, addr string) (net.Conn, error) {
				connHTTP, err := net.DialTimeout(netw, addr, time.Duration(timeout) * time.Second)
				if err != nil {
					log.WithFields(log.Fields{
						"error":	err.Error(),
					}).Error("dail timeout")
					return nil, err
				}
				connHTTP.SetDeadline(time.Now().Add(time.Second * 15))
				return connHTTP, nil

			},
			MaxIdleConnsPerHost:   64,
			ResponseHeaderTimeout: time.Millisecond * 20000,
			DisableKeepAlives:     false,
		},
	}

	jar, err := cookiejar.New(nil)
	if err != nil {
		panic(err)
	}
	connHTTP.Jar = jar

	return connHTTP
}

// initWebDriver for init web driver of firefox
func initWebDriver() *selenium.WebDriver {
	// Connect to the WebDriver instance running locally.
	fireCaps := firefox.Capabilities{}
	pre := make(map[string]interface{})
	pre["browser.tabs.remote.autostart"] = false
	pre["browser.privatebrowsing.autostart"] = true
	pre["browser.migrate.chrome.history.maxAgeInDays"] = 0
	pre["network.cookie.maxNumber"] = 3
	pre["browser.cache.memory.enable"] = false
	pre["browser.cache.disk.enable"] = false
	pre["browser.sessionhistory.max_total_viewers"] = 3
	fireCaps.Prefs = pre
	//caps := selenium.Capabilities{"browserName": "firefox"}
	caps := selenium.Capabilities{"browserName": "internet explorer"}
	caps.AddFirefox(fireCaps)
	
	addr := fmt.Sprintf(cm.SeleniumAddrPattern, cm.SeleniumPort)
	wd, err := selenium.NewRemote(caps, addr)
	if err != nil {
		log.WithFields(log.Fields{
			"error":	err.Error(),
		}).Panic("can not get selenium remote instance by initWebDriver")

		panic(err)
	}

	// set timeout
	wd.SetPageLoadTimeout(time.Duration(180) * time.Second)
	wd.SetAsyncScriptTimeout(time.Duration(180) * time.Second)
	
	log.Info("init web driver success...")

	return &wd
}

func (h *ServiceHTTP) restartWebDriver() {
	if h.wd != nil {
		err := (*h.wd).Quit()
		h.wd = nil
		if err != nil {
			log.WithFields(log.Fields{
				"error":	err.Error(),
			}).Error("quit browser failed by restartWebDriver")
		}
	}

	for {
		h.wd = initWebDriver()
		if h.wd != nil {
			log.Info("restart web driver success...")

			break
		}

		log.Error("restart web driver failed, keep trying")
		time.Sleep(time.Duration(5) * time.Second)
	}
}

// convertCustomResponse for convert response to custom response
func convertCustomResponse(resp *http.Response) *CustomResponse {
	cusResp := &CustomResponse{
		Headers: make(map[string]string),
	}

	for header := range resp.Header {
		cusResp.Headers[header] = resp.Header.Get(header)
	}
	cusResp.StatusCode = resp.StatusCode
	body, _ := ioutil.ReadAll(resp.Body)
	cusResp.Body = body

	return cusResp
}

// RequestPost for request post
func (h *ServiceHTTP) RequestPost(url string, body string, headers map[string]string) *CustomResponse {
	req, err := http.NewRequest("POST", url, bytes.NewBuffer([]byte(body)))
	if err != nil {
		log.WithFields(log.Fields{
			"url":	url,
		}).Error("can not new request at RequestPost")

		return nil
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	httpClient := getTransportHttpClient()
	resp, err := httpClient.Do(req)
	if err != nil {
		log.WithFields(log.Fields{
			"url":	url,
		}).Error("can not get body at RequestPost")

		return &CustomResponse{}
	}

	// count request num
	atomic.AddUint64(&h.RequestCounter, 1)

	return convertCustomResponse(resp)
}

// GetJsonRequestPost returns []byte by request post
func (h *ServiceHTTP) GetJsonRequestPost(pageURL string) []byte {
	u, _ := url.Parse(pageURL)
	index := strings.LastIndex(u.RequestURI(), "/")
	if index < 0 || index >= len(u.RequestURI()){
		log.WithFields(log.Fields{
			"pageURL":	pageURL,
		}).Warn("not a legal post url by GetJsonRequestPost")

		return nil
	}

	char := u.RequestURI()[strings.LastIndex(u.RequestURI(), "/")+1:]  // get url's last characteristic
	// compose to a post url
	postURL := fmt.Sprintf(cm.PostURLPattern, u.Scheme, strings.Replace(u.Host, "www", "api", 1))
	headers := make(map[string]string)
	headers["User-Agent"] = cm.HeaderUserAgent
	headers["content-type"] = cm.HeaderContentType
	headers["Accept"] = `application/json, text/plain, */*`
	resp := h.RequestPost(postURL, fmt.Sprintf("subdom=%s", char), headers)
	if resp == nil || resp.StatusCode != 200 {
		log.WithFields(log.Fields{
			"pageURL":	pageURL,
		}).Error("http request post page failed by GetJsonRequestPost")

		return nil
	}

	return resp.Body
}

// RequestGet return pointer of CustomResponse instance for request get
func (h *ServiceHTTP) RequestGet(url string, headers map[string]string) *CustomResponse {
	req, err := http.NewRequest("GET", url, strings.NewReader(""))
	if err != nil {
		log.WithFields(log.Fields{
			"url":	url,
		}).Error("can not new request by RequestGet")

		return nil
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.WithFields(log.Fields{
			"url":	url,
		}).Error("can not get body by RequestGet")

		return nil
	}

	defer resp.Body.Close()

	// count request num
	atomic.AddUint64(&h.RequestCounter, 1)

	return convertCustomResponse(resp)
}

// GetDocRequestGet returns doc pointer of goquery.Document instance by request get
func (h *ServiceHTTP) GetDocRequestGet(pageURL string) *goquery.Document {
	resp := h.RequestGet(pageURL, DefaultHeader())
	if resp == nil {
		return nil
	}
	if resp.StatusCode != 200 {
		log.WithFields(log.Fields{
			"statusCode":	resp.StatusCode,
			"body":			string(resp.Body),
		}).Error("request html error by GetDocRequestGet")

		return nil
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(resp.Body)))
	if err != nil {
		log.WithFields(log.Fields{
			"error":	err.Error(),
		}).Error("new document failed by GetDocRequestGet")

		return nil
	}

	return doc
}

// RequestWithTransportGet request get with transport
func (h *ServiceHTTP) RequestTransportGet(url string) *CustomResponse {
	req, err := http.NewRequest("GET", url, strings.NewReader(""))
	if err != nil {
		log.WithFields(log.Fields{
			"url":	url,
		}).Error("can not new request at RequestWithTransportGet")

		return nil
	}

	retry := 3
	resp := new(http.Response)
	var errD error
	httpClient := getTransportHttpClient()
	for i := 0; i < retry; i++ {
		resp, errD = httpClient.Do(req)
		if errD == nil && resp != nil && resp.StatusCode == 200 {
			break
		}
	}

	if errD != nil {
		log.WithFields(log.Fields{
			"url":	url,
		}).Error("can not get body at RequestWithTransportGet")

		return &CustomResponse{}
	}

	defer resp.Body.Close()

	// count request num
	atomic.AddUint64(&h.RequestCounter, 1)

	return convertCustomResponse(resp)
}

// DownloadImage returns success flag and image name
func (h *ServiceHTTP) DownloadImage(url string, dir string, rename string) (string, bool) {
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

	if ok, _ := ut.PathExists(dir); !ok {
		err := os.MkdirAll(dir, os.ModePerm)
		if err != nil {
			log.WithFields(log.Fields{
				"dir":		dir,
				"error":	err.Error(),
			}).Error("can not make dir by DownloadImage")

			return imageName, false
		}
	}

	filePath := fmt.Sprintf("%s/%s", dir, imageName)
	if ok, _ := ut.PathExists(filePath); ok {
		return imageName, true
	}

	// get response if not exist
	myResp := h.RequestTransportGet(url)
	if myResp == nil || len(myResp.Body) <= 0 {
		log.WithFields(log.Fields{
			"url":	url,
		}).Error("can not download this resource by DownloadImage")

		return "", false
	}

	file, err := os.Create(filePath)
	defer file.Close()

	if err != nil {
		log.WithFields(log.Fields{
			"filePath":	filePath,
			"error":	err.Error(),
		}).Error("can not create file by DownloadImage")

		return imageName, false
	}

	file.Write(myResp.Body)

	// debug
	log.WithFields(log.Fields{
		"imageName":	imageName,
	}).Debug("download image success")

	return imageName, true
}

// GetURLWebDriver for get specified url using web driver
func (h *ServiceHTTP) GetURLWebDriver(url string) *selenium.WebDriver {
	// debug
	last := time.Now()

	// https://blog.csdn.net/weixin_30906425/article/details/98371286
	now := time.Now()
	subTime := now.Sub(h.wdLastRestartTime)
	if subTime >= (time.Duration(24) * time.Hour) {
		log.Info("this browser has been running for more than one day do restart")
		h.wdLastRestartTime = now
		h.restartWebDriver()
	}

	if err := (*h.wd).Get(url); err != nil {
		log.WithFields(log.Fields{
			"url":   url,
			"error": err.Error(),
		}).Error("can not open url by web driver by GetURLWebDriver")

		return nil
	}

	// debug
	now = time.Now()
	log.WithFields(log.Fields{
		"time":	now.Sub(last),
	}).Debug("test duration, web driver get url by GetURLWebDriver")
	last = now

	return h.wd
}

// QuitWebDriver for quit web driver
func (h *ServiceHTTP) QuitWebDriver() {
	if (*h.wd) != nil {
		(*h.wd).Quit()
	}
}