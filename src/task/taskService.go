/*
  Package task for seek suggestion with specified page id
*/

package task

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/astaxie/beego"
	jsoniter "github.com/json-iterator/go"
	"github.com/micro/go-micro"
	"github.com/micro/go-micro/broker"
	"github.com/micro/go-micro/registry"
	"github.com/micro/go-micro/registry/etcd"
	"github.com/micro/go-micro/web"
	"github.com/micro/go-plugins/broker/nsq"
	log "github.com/sirupsen/logrus"

	cm "../common"
	ds "../deliveryService"
	hs "../httpService"
	mc "../mysqlClient"
	pb "../proto"
	sc "../scheduler"
	st "./sites"
)

// ServiceTask represents task service
type ServiceTask struct {
	SubChan         chan string
	ResChan			chan *cm.ProInfo
	PubChan 		chan string
	microService 	*micro.Service
	webService		*web.Service
	httpService   	*hs.ServiceHTTP
	db         		*mc.MySQLClient
	site			*st.SiteService
	delivery		*ds.ServiceDelivery
	SubCounter      uint64 // calculation receive num of subscriber, must use by atomic !!!
}

var instance *ServiceTask
var initTaskOnce sync.Once

// GetTaskInstance return ServiceTask pointer instance
func GetTaskInstance(useMicro bool, db ...*mc.MySQLClient) *ServiceTask {
	initTaskOnce.Do(func() {
		if len(db) > 0 {
			instance = new(ServiceTask)
			instance.init(useMicro, db[0])

			log.Info("init task service instance success...")
		} else {
			log.Fatal("first init task service do not get mysqlClient instance !!!")
		}
	})

	return instance
}

// init for init task service
func (t *ServiceTask) init(useMicro bool, db *mc.MySQLClient) {
	size := beego.AppConfig.DefaultInt("channelSize", cm.MaxChannelSize)
	t.SubChan = make(chan string, size)
	t.ResChan = make(chan *cm.ProInfo, size)
	t.PubChan = make(chan string, size)

	// init micro service
	if useMicro {
		t.microService = initMicroService()
		t.delivery = ds.GetDeliveryInstance(t.microService)

		// register process function to subscriber
		topic := beego.AppConfig.DefaultString("nsq::topic.sub", cm.TopicSUBName)
		channel := beego.AppConfig.DefaultString("nsq::queue", cm.ChannelName)
		t.delivery.RegisterSubscriberWithCh(process, topic, channel)
	}

	// init micro web service
	t.webService = t.initMicroWebService()

	t.db = db
	t.httpService = hs.GetHTTPInstance()
	t.site = st.GetSiteServiceInstance()

	atomic.StoreUint64(&t.SubCounter, 0) // init counter to 0
}

// initMicroWebService return pointer of micro web service
func (t *ServiceTask) initMicroWebService() *web.Service {
	// specified web service host
	ip := beego.AppConfig.DefaultString("micro::web.ip", cm.MicroWebAddress)
	port := beego.AppConfig.DefaultString("micro::web.port", cm.MicroWebPort)
	webHost := ip + ":" + port

	name := beego.AppConfig.DefaultString("micro::web.serviceName", cm.MicroWebServiceName)
	service := web.NewService(
		web.Name(name),
		web.Address(webHost),
	)

	service.HandleFunc("/foo", func(w http.ResponseWriter, request *http.Request) {
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

		atomic.AddUint64(&t.SubCounter, 1) // count receive num

		res := t.queryResource(resMap["url"], resMap["title"])
		response := fmt.Sprintf(`{"message": %v}`, res)

		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(response))
	})

	// Init will parse the command line flags.
	service.Init()

	return &service
}

// RunMicroWebService for run micro web service
func (t *ServiceTask) RunMicroWebService() {
	// Run web server
	if err := (*t.webService).Run(); err != nil {
		log.WithFields(log.Fields{
			"error info": err.Error(),
		}).Fatal("web service failed to run")
	}

	log.Info("exit micro web service success...")
}

// initMicroService return pointer of micro service
func initMicroService() *micro.Service {
	ip := beego.AppConfig.DefaultString("nsq::ip", cm.BrokerAddress)
	port := beego.AppConfig.DefaultString("nsq::port", cm.BrokerPort)
	var BrokerHosts = []string{ // specified nsq service host
		0: ip + ":" + port,
	}

	ip = beego.AppConfig.DefaultString("etcd::ip", cm.RegistryAddress)
	port = beego.AppConfig.DefaultString("etcd::port", cm.RegistryPort)
	var RegistryHosts = []string{ // specified etcd service host
		0: ip + ":" + port,
	}

	name := beego.AppConfig.DefaultString("micro::serviceName", cm.MicroServiceName)
	service := micro.NewService(
		// This name must match the package name given in your protobuf definition
		micro.Name(name),
		micro.Version("0.1"),
		micro.Broker(nsq.NewBroker(func(o *broker.Options) { // specified using nsq broker
			o.Addrs = BrokerHosts
		})),
		micro.Registry(etcd.NewRegistry(func(o *registry.Options) { // specified using etcd registry
			o.Addrs = RegistryHosts
		})),
	)

	// Init will parse the command line flags.
	service.Init()

	return &service
}

// RunMicroService for run micro service
func (t *ServiceTask) RunMicroService() {
	log.Info("start task service success...")

	// Run the server
	if err := (*t.microService).Run(); err != nil {
		log.WithFields(log.Fields{
			"error info": err.Error(),
		}).Fatal("micro service failed to run")
	}

	t.httpService.QuitWebDriver()

	log.Info("exit task service success...")
}

// process for subscriber function
func process(ctx context.Context, event *pb.Event) error {
	t := GetTaskInstance(true)
	if event != nil {
		t.SubChan <- event.GetMessage()

		atomic.AddUint64(&t.SubCounter, 1) // count receive num
	}

	// only for debug
	log.WithFields(log.Fields{
		"event": event,
	}).Debug("subscriber event")

	return nil
}

// checkResLegal returns true if the resource is legal
func checkResLegal(pi *cm.ProInfo) bool {
	if pi != nil && len(pi.Cover) > 0 && len(pi.Desc) > 0 {
		return true
	} else {
		return false
	}
}

// TaskQueryResource for get site resource by pageURL
func (t *ServiceTask) TaskQueryResource(data *sc.DataBlock) {
	resTitle := data.Extra.(string)
	pageURL := data.Message.(string)

	t.queryResource(pageURL, resTitle)
}

// TaskParseURL for parse landing URL
func (t *ServiceTask) TaskParseURL(data *sc.DataBlock) {
	pageURL := data.Message.(string)

	t.parseWebPage(pageURL)
}