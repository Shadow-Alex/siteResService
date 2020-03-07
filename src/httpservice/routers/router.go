/*
  Package routers for web request
*/

package routers

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"siteResService/src/data"
	"sync"
	"sync/atomic"

	"github.com/astaxie/beego"
	jsoniter "github.com/json-iterator/go"
	log "github.com/sirupsen/logrus"

	cm "siteResService/src/common"
	tk "siteResService/src/taskservice"
)

var instance *Router
var task *tk.TaskService
var initRouterOnce sync.Once

// Router represents Router
type Router struct {
	// mapping route with func
	RouterMap	map[string]func(w http.ResponseWriter, request *http.Request)
	subCounter	*uint64
}

// GetRouters return pointer of Router instance
func GetRouters(t *tk.TaskService, subCounter *uint64) *Router {
	initRouterOnce.Do(func() {
		if t == nil {
			log.Panic("can not get task instance")
		}

		instance = new(Router)
		task = t
		instance.subCounter = subCounter
		instance.initMapping()
	})

	return instance
}

// initMapping for init route and func mapping
func (r *Router) initMapping() {
	version := beego.AppConfig.DefaultString("version", cm.Version)
	r.RouterMap = make(map[string]func(w http.ResponseWriter, request *http.Request))
	r.RouterMap["/" + version + "/siteResource"] = getSiteResource

	// lijing
	r.RouterMap["/" + version + "/import"] = data.ImportData
}

// add route
func (r *Router) AddRouter(router string,
	f func(w http.ResponseWriter, request *http.Request)) error {
	if r.RouterMap == nil {
		return errors.New("router map is null")
	}

	r.RouterMap[router] = f
	return nil
}

// define func
var getSiteResource = func(w http.ResponseWriter, request *http.Request) {
	body, err := ioutil.ReadAll(request.Body)
	if err != nil {
		log.WithFields(log.Fields{
			"error":	err.Error(),
		}).Error("can not get post params by micro web service")

		return
	}

	resMap := make(map[string]string)
	if err := jsoniter.Unmarshal(body, &resMap); err != nil{
		log.WithFields(log.Fields{
			"error":	err.Error(),
		}).Error("can not Unmarshal post params to map by micro web service")

		return
	}

	atomic.AddUint64(instance.subCounter, 1) // count receive num

	res := task.QueryResource(resMap["url"], resMap["title"])
	response := fmt.Sprintf(`{"message": %v}`, res)

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(response))
}