package common

import (
	"github.com/astaxie/beego/orm"
)

const (
	// RunTypeMicro for running service with micro
	RunTypeMicro = "micro"

	// RunTypeStandAlone for running service alone
	RunTypeStandAlone = "standalone"

	// DestStandAloneDB for running service alone and get page id from db
	DestStandAloneDB = "db"
	// siteResourceFile for name of site resource file
	SiteResourceFile = "siteResource.csv"
	// SiteSpecFile for name of specifications file
	SiteSpecFile = "siteSpecifications.csv"
	// SiteSetFile for name of set meal file
	SiteSetFile = "siteSetMeal.csv"

	// HTMLFormat for distinguish the method of parsing page by using html
	HTMLFormat = "html"
	// WebFormat for distinguish the method of parsing page by web driver
	WebFormat = "web"
	// JSONFormat for distinguish the method of parsing page by using json
	JSONFormat = "json"
	// LabelSeparate for separate each cascade label
	LabelSeparate = "|"
	// ListSeparate indicates that previous label in front of ";" is a list
	ListSeparate = ";"

	// IdleRunFromDate for run data from date, if yesterday's data finished
	IdleRunFromDate = "1970-01-01"

	// SuperviseGap for time gap of supervise
	SuperviseGap = 3

	// PprofPort pprof plugin for monitor memory use
	PprofPort = "6064"

	// MaxChannelSize for max channel size
	MaxChannelSize = 10

	// ProductionENV for whether use product environment
	ProductionENV = false

	// ImageDIR for image dir
	ImageDir = "./imageResource/"
	// ImagePrefixUploads for image prefix uploads
	ImagePrefixUploads = "./imageResource/"
	// ImagePrefixDefault for image prefix default value
	ImagePrefixDefault = "~/penghu/project/go/siteResService"

	// T1ServerURL
	T1ServerURL = `https://cpc.dotact365.com/sale?id=`
	// T1CND
	T1CND = `https://d3jd93afziw2li.cloudfront.net/`

	// LogDir for logs dir
	LogDir = "./logs"
	// LogFilename for log file name
	LogFilename = "service.log"
	// LogLevel for log output level:  Trace=6; Debug=5; Info=4; Warn=3; Error=2; Fatal=1; Panic=0
	LogLevel = 5
	// LogWithMaxAge for log with max age (keep time, unit: 24hour)
	LogWithMaxAge = 20

	// SchedulerChannelNum for scheduler channel number
	SchedulerChannelNum = 5
	// SchedulerTaskQueueSize for scheduler task queue size
	SchedulerTaskQueueSize = 10

	// MicroServiceName for micro service name
	MicroServiceName = "go.micro.service"
	// MicroWebServiceName for micro web service name
	MicroWebServiceName = "go.micro.web.service"
	// MicroWebAddress for micro web service ip
	MicroWebAddress = "localhost"
	// MicroWebPort for micro web service port
	MicroWebPort = "8099"
	// BrokerAddress for micro broker address using nsq
	BrokerAddress = "localhost"
	// BrokerPort for micro broker port using nsq
	BrokerPort = "4150"
	// RegistryAddress for micro registry address using etcs
	RegistryAddress = "localhost"
	// RegistryPort for micro registry port using etcs
	RegistryPort = "2379"
	// TopicSUBname for nsq receive topic
	TopicSUBName = "zfky.topic.service"
	// TopicPUBName for nsq send topic
	TopicPUBName = "zfky.topic.client"
	// ChannelName for nsq channel name
	ChannelName = "queue.service"

	// RedisIP for redis ip
	RedisIP = "localhost"
	// RedisPort for redis port
	RedisPort = "9527"
	// RedisPass for redis pass
	RedisPass = "zfky!"
	// RedisPartition for redis partition
	RedisPartition = 0

	// DBMaxIdleCONNS FOR mysql max idle connections
	DBMaxIdleCONNS = 10
	// DBMaxOpenCONNS for  mysql max idle connections
	DBMaxOpenCONNS = 60
	// DBMultiInsertSize for mysql multi insert items num
	DBMultiInsertSize = 20
	// DBLimit for result limit of one query
	DBQueryLimit = 1000
	// DBQueryGap between two query operation for slow down http request frequency
	DBQueryGap = 10
	// DBRetryCount for retry when operation db failed
	DBRetryCount = 3
	// DBRetryDelay for retry delay time
	DBRetryDelay = 1

	// CookieUserAgent for cookie header  user agent
	HeaderUserAgent = `Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/78.0.3904.70 Safari/537.36`
	HeaderContentType = `application/x-www-form-urlencoded`
	// PostURLPattern for url pattern which request post to get json
	PostURLPattern = `%s://%s/product/product/index`
	// HTTPTimeOut for http request timeout
	HTTPTimeOut = 5

	// SeleniumAddrPattern for webdriver address pattern
	//SeleniumAddrPattern = `http://localhost:%d/wd/hub`
	SeleniumAddrPattern = `http://192.168.3.9:%d/wd/hub`
	// seleniumPort for web driver port
	SeleniumPort = 8083
	// SeleniumPath for path of selenium-server-standalone
	SeleniumPath = "vendor/selenium-server-standalone-3.141.59.jar"
	// GeckoDriverPath for path of geckodriver
	GeckoDriverPath = "vendor/geckodriver"
)

// PubInfo represents info which publisher need
type PubInfo struct {
	Topic string
	Magic int64
}

// DBUpdateInfo represents info which need update
type DBUpdateInfo struct {
	TableName	string
	Condition	*orm.Condition
	Params		*orm.Params
}

// ImageInfo represents landing page image info
type ImageInfo struct {
	Iid      string `json:"iid"`
	Gid      string `json:"gid"`
	URL      string `json:"url"`
	Title    string `json:"title"`
	Original string `json:"original"`
	Thumb    string `json:"thumb"`
	Md5      string `json:"md5"`
	RootPath string `json:"root_path"`
	IsImage  bool   `json:"is_image"`
	IsCover  bool   `json:"is_cover"`
}

// ProInfo represents landing page info
type ProInfo struct {
	PageURL 	string
	Cover   	[]ImageInfo  // head image
	Title   	string
	Price   	string
	Desc    	string  // descriptions with image
	Spec    	[][]string  // specifications with image
	Set			[][]string  // set meal
	//Images  	[]ImageInfo
	Template	string
}

// LabelsParse represents label required for parsing page
type LabelsParse struct {
	Character	string		// for distinguish the method of parsing page
	Order		[]string		// for order label
	Cover 		[]string	// head image label
	Title		[]string
	Price		[]string
	Desc		[]string
	Spec		[]string
	Set			[]string
}

// CargoExtInfo represents ext info
type CargoExtInfo struct {
	ID			int64
	CargoID		uint
	LandingURL	string
}

// DownLoadInfo represents info which download required
type DownLoadInfo struct {
	URL		string
	Name	string
}

// GetDBConns for get db connection string
//func GetDBConns(table string) string {
//	env := beego.AppConfig.DefaultBool("productionENV", ProductionENV)
//	if env {
//		switch(table) {
//		case "dbKR":
//			return DBKRConnsProduct
//		case "dbWC":
//			return DBWCConnsProduct
//		}
//	} else {
//		switch(table) {
//		case "dbKR":
//			return DBKRConnsTest
//		case "dbWC":
//			return DBWCConnsTest
//		}
//	}
//
//	return ""
//}