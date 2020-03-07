/*
  Package task for seek suggestion with specified page id
*/

package taskservice

import (
	"sync"

	"github.com/astaxie/beego"
	log "github.com/sirupsen/logrus"

	cm "siteResService/src/common"
	hs "siteResService/src/httpservice"
	mc "siteResService/src/mysqlclient"
	sc "siteResService/src/scheduler"
	st "siteResService/src/taskservice/sites"
)

// TaskService represents task service
type TaskService struct {
	ResChan			chan *cm.ProInfo
	PubChan 		chan string
	httpService   	*hs.ServiceHTTP
	db         		*mc.MySQLClient
	site			*st.SiteService
}

var instance *TaskService
var initTaskOnce sync.Once

// GetTaskInstance return TaskService pointer instance
func GetTaskInstance(db ...*mc.MySQLClient) *TaskService {
	initTaskOnce.Do(func() {
		if len(db) > 0 {
			instance = new(TaskService)
			instance.init(db[0])

			log.Info("init task service instance success...")
		} else {
			log.Fatal("first init task service do not get mysqlclient instance !!!")
		}
	})

	return instance
}

// init for init task service
func (t *TaskService) init(db *mc.MySQLClient) {
	size := beego.AppConfig.DefaultInt("channelSize", cm.MaxChannelSize)
	t.ResChan = make(chan *cm.ProInfo, size)
	t.PubChan = make(chan string, size)

	t.db = db
	t.httpService = hs.GetHTTPInstance()
	t.site = st.GetSiteServiceInstance()
}

// TaskQueryResource for get site resource by pageURL
func (t *TaskService) TaskQueryResource(data *sc.DataBlock) {
	resTitle := data.Extra.(string)
	pageURL := data.Message.(string)

	t.QueryResource(pageURL, resTitle)
}

// TaskParseURL for parse landing URL
func (t *TaskService) TaskParseURL(data *sc.DataBlock) {
	pageURL := data.Message.(string)

	t.parseWebPage(pageURL)
}