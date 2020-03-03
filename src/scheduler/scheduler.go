/*
  Package scheduler use channel to schedule task.
*/

package scheduler

import (
	"sync"
	"time"
	"math/rand"

	"github.com/astaxie/beego"
	"github.com/gammazero/workerpool"
	log "github.com/sirupsen/logrus"

	cm "../common"
)

// DataBlock represents data block which contains message and extra info
type DataBlock struct {
	Extra   interface{}
	Message interface{}
}

// ControlInfo represents task number, if used, Name and CtrNum should not be nil
type ControlInfo struct {
	Name		string  // control name
	CtrlNum		int  // control all task number
}

// Task represents basic unit to be dispatched.
type Task struct {
	CtrlInfo	*ControlInfo  // for control task
	Data   		*DataBlock  // data to do in the task
	DoTask 		func(*DataBlock)  // do task function
}

// dispatcher represents  one dispatcher of scheduler
type dispatcher struct {
	ctrlQueueNum	chan bool  // control the num of input into routine pool
	channel 		chan Task
	pool    		*workerpool.WorkerPool
}

// Scheduler represents scheduler struct.
type Scheduler struct {
	name       			string
	dispatcherMap 		sync.Map  // for dispatch all except ctrl task
	dispatcherCtrlMap	sync.Map  // for need control dispatch task number
}

var scheduler *Scheduler
var once sync.Once
var dispatcherNumber int
var taskQueueSize int

// GetScheduler returns Scheduler instance pointer
func GetScheduler() *Scheduler {
	once.Do(func() {
		rand.Seed(time.Now().UnixNano())

		scheduler = newScheduler()
	})

	return scheduler
}

// initDispatcherMap for Initialization dispatchers
func initDispatcherMap(s *Scheduler) {
	for i := 0; i < dispatcherNumber; i++ {
		d := new(dispatcher)
		d.channel = make(chan Task, 1)
		d.pool = workerpool.New(taskQueueSize)
		s.dispatcherMap.Store(i, d)
	}
}

// newScheduler return Scheduler instance pointer which constructed
func newScheduler() *Scheduler {
	dispatcherNumber = beego.AppConfig.DefaultInt("scheduler::channelNum", cm.SchedulerChannelNum)
	taskQueueSize = beego.AppConfig.DefaultInt("scheduler::taskQueueSize", cm.SchedulerTaskQueueSize)

	scheduler = new(Scheduler)

	scheduler.name = "scheduler1"
	//scheduler.dispatcherMap = make(map[int]*dispatcher)

	initDispatcherMap(scheduler)
	startGoList(scheduler)

	log.WithFields(log.Fields{
		"name":				"scheduler1",
		"dispatcherNum":	dispatcherNumber,
		"poolNum":			taskQueueSize,
	}).Info("scheduler init success...")

	return scheduler
}

// initDispatcherCtrlMap for Initialization dispatcherCtrls
func (s *Scheduler) initDispatcherCtrlMap(name string, num int) *dispatcher {
	d := new(dispatcher)
	d.ctrlQueueNum = make(chan bool, num)  // control the num of input into routine pool
	d.channel = make(chan Task, 1)
	d.pool = workerpool.New(num)
	s.dispatcherCtrlMap.Store(name, d)

	// make dispatcher running
	go func(d *dispatcher) {
		for {
			select {
			case task := <-d.channel:
				d.ctrlDoTask(task)
			}
		}
	}(d)

	log.WithFields(log.Fields{
		"name":		name,
		"poolNum":	num,
	}).Info("dispatcher control init success...")

	return d
}

// startGoList for start go channel list
func startGoList(s *Scheduler) {
	// TODO how to confirm startGoList finished
	for i := 0; i < dispatcherNumber; i++ {
		d, ok := s.dispatcherMap.Load(i)
		if !ok {
			log.WithFields(log.Fields{
				"index":	i,
			}).Error("do not get dispatcher")

			continue
		}
		go func(d *dispatcher) {
			for {
				select {
				case task := <-d.channel:
					d.doTask(task)
				}
			}
		}(d.(*dispatcher))
	}

	log.Info("all scheduler task channels has started...")
}

// doTask for run task DoTask function by routine pool
func (d *dispatcher) doTask(task Task) {
	t := task
	d.pool.Submit(func() {
		if t.DoTask != nil {
			t.DoTask(t.Data)
		}
	})
}

// ctrlDoTask for run task DoTask function by controlled routine pool num
func (d *dispatcher) ctrlDoTask(task Task) {
	d.ctrlQueueNum <- true

	t := task
	ctrlQueue := &(d.ctrlQueueNum)
	d.pool.Submit(func() {
		if t.DoTask != nil {
			t.DoTask(t.Data)
		}

		<- *ctrlQueue
	})
}

// GetPoolWaitingQueueSize for pool waitingQueue size, use for debug
func (s *Scheduler) GetPoolWaitingQueueSize() int {
	var counter int
	for i := 0; i < dispatcherNumber; i++ {
		d, ok := s.dispatcherMap.Load(i)
		if ok {
			counter += d.(*dispatcher).pool.WaitingQueueSize()
		}
	}

	return counter
}

// GetCtrlPoolWaitingQueueSize for control pool waitingQueue size, use for debug
func (s *Scheduler) GetCtrlPoolWaitingQueueSize(name string) int {
	var counter int
	d, ok := s.dispatcherCtrlMap.Load(name)
	if ok {
		counter += d.(*dispatcher).pool.WaitingQueueSize()
	}

	return counter
}

// TODO: need to be think, how to effectively select the channel with the largest space
// AddTask return true if adding task to a random channel success
func (s *Scheduler) AddTask(task Task) {
	ctrl := task.CtrlInfo
	if ctrl == nil {  // if task do not need  control running number, use this dispatcher
		rand.Seed(time.Now().UnixNano())
		index := rand.Intn(dispatcherNumber)
		d, ok := s.dispatcherMap.Load(index%dispatcherNumber)
		if ok {
			d.(*dispatcher).channel <- task  //if this scheduler channel is full, then block, for not process more data
		}
	} else {  // if task is need control running number, use this specified dispatcherCtrl
		d, ok := s.dispatcherCtrlMap.Load(ctrl.Name)
		if !ok {
			d = s.initDispatcherCtrlMap(ctrl.Name, ctrl.CtrlNum)
		}
		d.(*dispatcher).channel <- task  //if this scheduler channel is full, then block, for not process more data
	}
}
