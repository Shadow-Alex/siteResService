/*
  Package microservice using micro broker's publisher and subscriber to send and receive event.
*/

package microservice

import (
	"time"
	"context"
	"strconv"
	"sync/atomic"
	"runtime/debug"

	"github.com/micro/go-micro"
	"github.com/micro/go-micro/server"
	log "github.com/sirupsen/logrus"

	cm "siteResService/src/common"
	pb "siteResService/src/proto"
	sc "siteResService/src/scheduler"
	ut "siteResService/src/util"
)

// ServiceDelivery represents delivery service to send and receive event.
type ServiceDelivery struct {

}

// create publisher instance with specified topic.
func (m *MicroService) createPublisher(topic string) *micro.Publisher {
	pub := micro.NewPublisher(topic, (*instance.microService).Client())

	return &pub
}

// getPublisher get publisher instance with specified topic.
func (m *MicroService) getPublisher(topic string) *micro.Publisher {
	pub, ok := m.pubSMap.Load(topic)
	if !ok {
		pub = m.createPublisher(topic)
		if pub == nil {
			log.WithFields(log.Fields{
				"topic": topic,
			}).Error("get publisher with specified topic failed !")

			return nil
		}

		m.pubSMap.Store(topic, pub.(*micro.Publisher))
	}

	return pub.(*micro.Publisher)
}

// generateEvent return list of pb.Event pointer created by given suggestion id list
func generateEvent(magic int64, msg string) *pb.Event {
	return &pb.Event{ // make delivery msg as a event which defined at proto file
		Id:        ut.GetUUID(),
		Timestamp: time.Now().Unix(),
		Magic:     magic,
		Message:   msg,
	}
}

// SendMsgWithTopic send message to specified topic by publisher.
func (m *MicroService) SendMsgWithTopic(topic string, magic int64, msg string) {
	publisher := m.getPublisher(topic)
	if publisher != nil {
		for { // continue to retry publish message until success
			ev := generateEvent(magic, msg)
			if err := (*publisher).Publish(context.Background(), ev); err != nil {
				time.Sleep(time.Duration(5) * time.Second) // if publish failed, wait for 5s, to retry

				log.WithFields(log.Fields{
					"topic":      topic,
					"event":      ev,
					"error info": err.Error(),
				}).Error("publish event to topic failed, wait for 5s to retry...")
			} else {
				atomic.AddUint64(&m.DeliverCounter, 1) // count deliver num

				// only for debug
				log.WithFields(log.Fields{
					"event": ev,
				}).Debug("publish event")

				break // if publish success, break for
			}
		}
	}
}

// TaskSend send message using scheduler DataBlock.
func (m *MicroService) TaskSend(data *sc.DataBlock) {
	defer func() { // add recover to catch panic
		if err := recover(); err != nil {
			log.WithFields(log.Fields{
				"error info": err,
			}).Fatal("delivery send event failed") // err is panic incoming content of panic
			log.Fatal(string(debug.Stack()))
		}
	}()

	pubInfo := data.Extra.(cm.PubInfo) //transfer to cm.PubInfo type for get topic and magic
	m.SendMsgWithTopic(pubInfo.Topic, pubInfo.Magic, strconv.Itoa(data.Message.(int)))
}

// RegisterSubscriber return false if register subscriber receive process function to specified topic failed.
func (m *MicroService) RegisterSubscriber(function interface{}, topic string) bool {
	if function == nil {
		log.Error("can not register subscriber, for process function is nil !")

		return false
	}
	if len(topic) <= 0 {
		log.Error("can not register subscriber, for specified topic is empty !")

		return false
	}

	err := micro.RegisterSubscriber(topic, (*m.microService).Server(), function)

	if err != nil {
		log.WithFields(log.Fields{
			"topic":      topic,
			"error info": err.Error(),
		}).Error("register receive process function to topic failed !")

		return false
	}

	log.WithFields(log.Fields{
		"topic": topic,
	}).Info("register receive process function to topic success...")

	return true
}

// RegisterSubscriberWithCh return false if register subscriber receive process function to specified topic and channel failed.
func (m *MicroService) RegisterSubscriberWithCh(function interface{}, topic string, channel string) bool {
	if function == nil {
		log.Error("can not register subscriber, for process function is nil !")

		return false
	}
	if len(topic) <= 0 {
		log.Error("can not register subscriber, for specified topic is empty !")

		return false
	}

	// register subscriber with queue, each message is delivered to a unique subscriber
	err := micro.RegisterSubscriber(topic, (*m.microService).Server(), function, server.SubscriberQueue(channel)) // specified a channel name
	if err != nil {
		log.WithFields(log.Fields{
			"topic":      topic,
			"channel":    channel,
			"error info": err.Error(),
		}).Error("register subscriber receive process function to specified topic and channel failed !")

		return false
	}

	log.WithFields(log.Fields{
		"topic":   topic,
		"channel": channel,
	}).Info("register subscriber receive process function to specified topic and channel success...")

	return true
}