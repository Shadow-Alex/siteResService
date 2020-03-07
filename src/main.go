/*
  Package main, program entry.
*/

package main

import (
	"net/http"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/astaxie/beego"
	log "github.com/sirupsen/logrus"

	cm "siteResService/src/common"
	hs "siteResService/src/httpservice"
	rt "siteResService/src/httpservice/routers"
	ms "siteResService/src/microservice"
	mc "siteResService/src/mysqlclient"
	sc "siteResService/src/scheduler"
	sa "siteResService/src/standalone"
	tk "siteResService/src/taskservice"
	ut "siteResService/src/util"
)


// Server represents main server
type Server struct {
	micro			*ms.MicroService
	standalone  	*sa.StandAlone
	task      		*tk.TaskService
	scheduler 		*sc.Scheduler
	http      		*hs.ServiceHTTP
	delivery  		*ms.ServiceDelivery
	db   			*mc.MySQLClient
	subChan			chan string
	subCounter		uint64  // calculation receive num of subscriber, must use by atomic !!!
}

var initOnce sync.Once
var server *Server

// initCommonRes for init common resource
func initCommonRes() {
	// sets the maximum number of CPUs that can be executing simultaneously and returns the previous setting
	runtime.GOMAXPROCS(runtime.NumCPU())

	// init logrus
	logHook := ut.InitLogrus()
	if logHook != nil {
		log.AddHook(logHook)
	}

	// open pprof, listen request
	port := beego.AppConfig.DefaultString("pprof.port", cm.PprofPort)
	go func(port string) {
		ip := "0.0.0.0:" + port
		if err := http.ListenAndServe(ip, nil); err != nil {
			log.WithFields(log.Fields{
				"ip":	ip,
			}).Info("start pprof failed")
			os.Exit(1)
		}
	}(port)
}

// supervise for supervise receive, delivery, db operation speed, per second
func supervise() {
	tdur := beego.AppConfig.DefaultInt("supervise.gap", cm.SuperviseGap)
	for {
		subNum := atomic.LoadUint64(&server.subCounter)
		pubNum := atomic.LoadUint64(&server.micro.DeliverCounter)
		requestNum := atomic.LoadUint64(&server.http.RequestCounter)
		//insertDBNum := atomic.LoadUint64(&server.db.InsertCounter)
		//updateDBNum := atomic.LoadUint64(&server.db.UpdateCounter)

		// reset counter to 0
		atomic.StoreUint64(&server.subCounter, 0)
		atomic.StoreUint64(&server.micro.DeliverCounter, 0)
		atomic.StoreUint64(&server.http.RequestCounter, 0)
		//atomic.StoreUint64(&server.db.InsertCounter, 0)
		//atomic.StoreUint64(&server.db.UpdateCounter, 0)

		log.WithFields(log.Fields{
			"sub":  	subNum / uint64(tdur),
			"pub":     	pubNum / uint64(tdur),
			"request":  requestNum / uint64(tdur),
			//"insertDB": insertDBNum / uint64(tdur),
			//"updateDB": updateDBNum / uint64(tdur),
			"routine":  runtime.NumGoroutine(),
		}).Info("running condition (per second)")

		time.Sleep(time.Duration(tdur) * time.Second)
	}
}

// startMicroServer for start main server
func startMicroServer() {
	initOnce.Do(func() {
		server = new(Server)

		atomic.StoreUint64(&server.subCounter, 0) // init counter to 0
		size := beego.AppConfig.DefaultInt("channelSize", cm.MaxChannelSize)
		server.subChan = make(chan string, size)
		server.scheduler = sc.GetScheduler()
		server.http = hs.GetHTTPInstance()
		//conn := cm.GetDBConns("KR")  // get db connection
		//if len(conn) <= 0 {
		//	log.Fatal("can not get db connection, exit")
		//
		//	return
		//}
		//server.db = mc.GetMySQLClientInstance(conn)
		//GetTaskServiceInstance will create micro service instance, should before GetDeliveryServiceInstance
		server.task = tk.GetTaskInstance(server.db)

		// init micro service
		server.micro = ms.GetMicroService(rt.GetRouters(server.task, &server.subCounter),
			server.http, &server.subChan, &server.subCounter)

		// debug, for temporary
		server.standalone = sa.GetStandAloneInstance(server.db, &server.subChan, &server.subCounter)

		go supervise()  // supervise speed

		go dispatch(server)  // dispatch msg

		go server.micro.RunMicroService()  // go routine run micro service as main process

		server.micro.RunMicroWebService()  // run micro web service
	})
}

// dispatch for dispatch task
func dispatch(server *Server) {
	for {
		select {
		// do task of parse url
		case msg := <-server.subChan:
			ctrl := &sc.ControlInfo{
				Name:    "http",  // must has value
				CtrlNum: 30,  // the size of concurrent routine pool
			}
			data := &sc.DataBlock{
				Extra:   nil,
				Message: msg,
			}
			server.scheduler.AddTask(sc.Task{
				CtrlInfo:	ctrl,
				Data:   	data,
				DoTask: 	server.task.TaskParseURL,
			})
		// do task of save site resource
		case msg := <-server.task.ResChan:
			data := &sc.DataBlock{
				Extra:   nil,
				Message: msg,
			}
			server.scheduler.AddTask(sc.Task{
				CtrlInfo:	nil,
				Data:   	data,
				DoTask: 	server.standalone.TaskSaveResultToFile,
			})
		}
	}
}

// startStandAloneServer for start main server
func startStandAloneServer(destSCR string) {
	initOnce.Do(func() {
		server = new(Server)

		atomic.StoreUint64(&server.subCounter, 0) // init counter to 0
		size := beego.AppConfig.DefaultInt("channelSize", cm.MaxChannelSize)
		server.subChan = make(chan string, size)
		server.scheduler = sc.GetScheduler()
		server.http = hs.GetHTTPInstance()
		//conn := cm.GetDBConns("dbWC")  // get db connection
		//if len(conn) <= 0 {
		//	log.Fatal("can not get db connection, exit")
		//
		//	return
		//}
		//// must have one register DataBase alias named `default` !!!
		//server.db = mc.GetMySQLClientInstance(conn)

		// GetStandAloneInstance will use task, GetTaskInstance should before GetStandAloneInstance
		server.task = tk.GetTaskInstance(server.db)

		// init micro service
		server.micro = ms.GetMicroService(rt.GetRouters(server.task, &server.subCounter),
			server.http, &server.subChan, &server.subCounter)

		server.standalone = sa.GetStandAloneInstance(server.db, &server.subChan, &server.subCounter)

		//go supervise() // supervise status

		// two ways of running
		if destSCR == cm.DestStandAloneDB {  // for using db to get page id
			go server.standalone.GetProsFromDB()
			//go server.task.ResetService()
		} else {  // for using file to get csv file
			go server.standalone.GetPageURLFromFile(destSCR)
		}

		// not use go routine for block main process
		go dispatchStandAlone(server)

		server.micro.RunMicroWebService()

		server.standalone.CloseFileTGT()
	})
}

// dispatchStandAlone for dispatch task
func dispatchStandAlone(server *Server) {
	for {
		select {
		// do task of parse url
		case msg := <-server.subChan:
			ctrl := &sc.ControlInfo{
				Name:    "http",  // must has value
				CtrlNum: 30,  // the size of concurrent routine pool
			}
			data := &sc.DataBlock{
				Extra:   nil,
				Message: msg,
			}
			server.scheduler.AddTask(sc.Task{
				CtrlInfo:	ctrl,
				Data:   	data,
				DoTask: 	server.task.TaskParseURL,
			})
		// do task of save site resource
		case msg := <-server.task.ResChan:
			data := &sc.DataBlock{
				Extra:   nil,
				Message: msg,
			}
			server.scheduler.AddTask(sc.Task{
				CtrlInfo:	nil,
				Data:   	data,
				DoTask: 	server.standalone.TaskSaveResultToFile,
			})
		}
	}
}

// main function
func main() {
	initCommonRes()

	//runType := cm.RunTypeStandAlone
	//if len(os.Args) > 3 {
	//	log.Error("do not get args for source file, exit")

	//
	//	return
	//}
	//destSCR := os.Args[2]
	//
	//switch runType {
	//case cm.RunTypeStandAlone:
	//	startStandAloneServer(destSCR)
	//case cm.RunTypeMicro:
	//	startMicroServer()
	//}

	runType := cm.RunTypeMicro
	destSCR := cm.DestStandAloneDB
	if len(os.Args) > 1 {
		runType = os.Args[1]
	}
	if len(os.Args) > 2 {
		destSCR = os.Args[2]
	}

	switch runType {
	case cm.RunTypeStandAlone:
		startStandAloneServer(destSCR)

	case cm.RunTypeMicro:
		startMicroServer()
	}
}
