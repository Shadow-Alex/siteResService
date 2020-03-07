/*
  Package microservice for micro service
*/

package microservice

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/astaxie/beego"
	"github.com/micro/go-micro"
	"github.com/micro/go-micro/broker"
	"github.com/micro/go-micro/registry"
	"github.com/micro/go-micro/registry/etcd"
	"github.com/micro/go-micro/web"
	"github.com/micro/go-plugins/broker/nsq"
	log "github.com/sirupsen/logrus"

	cm "siteResService/src/common"
	hs "siteResService/src/httpservice"
	rt "siteResService/src/httpservice/routers"
	pb "siteResService/src/proto"
)

var instance *MicroService
var initMicroOnce sync.Once

// MicroService represents micro service
type MicroService struct {
	microService 	*micro.Service
	router			*rt.Router
	webService		*web.Service
	httpService		*hs.ServiceHTTP
	pubSMap        	sync.Map       // store publisher map, publisher associated with topic
	subChan			*chan string
	subCounter		*uint64
	DeliverCounter 	uint64         // calculation num of delivery by publisher, must use by atomic !!!
}

// GetMicroService return pointer of MicroService instance
func GetMicroService(router *rt.Router, http *hs.ServiceHTTP, subChan *chan string, subCounter *uint64) *MicroService {
	initMicroOnce.Do(func() {
		instance = new(MicroService)

		instance.router = router
		instance.httpService = http
		instance.subChan = subChan
		instance.subCounter = subCounter

		// useMicro for init micro service
		useMicro := beego.AppConfig.DefaultBool("nsq::use.micro", cm.UseMicro)
		if useMicro {
			instance.microService = initMicro()

			// register process function to subscriber, should after init micro
			topic := beego.AppConfig.DefaultString("nsq::topic.sub", cm.TopicSUBName)
			channel := beego.AppConfig.DefaultString("nsq::queue", cm.ChannelName)
			instance.RegisterSubscriberWithCh(process, topic, channel)
		}
		// useWeb for init web service
		useWeb := beego.AppConfig.DefaultBool("nsq::use.web", cm.UseWeb)
		if useWeb {
			instance.webService = initMicroWeb()
		}
	})

	return instance
}

// initMicroWeb return pointer of micro web service
func initMicroWeb() *web.Service {
	// specified web service host
	ip := beego.AppConfig.DefaultString("micro::web.ip", cm.MicroWebAddress)
	port := beego.AppConfig.DefaultString("micro::web.port", cm.MicroWebPort)
	webHost := ip + ":" + port

	name := beego.AppConfig.DefaultString("micro::web.serviceName", cm.MicroWebServiceName)
	service := web.NewService(
		web.Name(name),
		web.Address(webHost),
	)

	// add all routes
	for key, value := range instance.router.RouterMap {
		service.HandleFunc(key, value)
	}

	// Init will parse the command line flags.
	service.Init()

	return &service
}

// RunMicroWebService for run micro web service
func (m *MicroService) RunMicroWebService() {
	if m.webService == nil {
		log.Error("can not run web service, for not init")

		return
	}

	// Run web server
	if err := (*m.webService).Run(); err != nil {
		log.WithFields(log.Fields{
			"error info": err.Error(),
		}).Fatal("web service failed to run")
	}

	log.Info("exit micro web service success...")
}

// initMicro return pointer of micro service
func initMicro() *micro.Service {
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
func (m *MicroService) RunMicroService() {
	if m.microService == nil {
		log.Error("can not run micro service, for not init")

		return
	}

	log.Info("start micro service success...")

	// Run the server
	if err := (*m.microService).Run(); err != nil {
		log.WithFields(log.Fields{
			"error info": err.Error(),
		}).Fatal("micro service failed to run")
	}

	m.httpService.QuitWebDriver()

	log.Info("exit task service success...")
}

// process for subscriber function
func process(ctx context.Context, event *pb.Event) error {
	if event != nil {
		*instance.subChan <- event.GetMessage()

		atomic.AddUint64(instance.subCounter, 1) // count receive num
	}

	// only for debug
	log.WithFields(log.Fields{
		"event": event,
	}).Debug("subscriber event")

	return nil
}
