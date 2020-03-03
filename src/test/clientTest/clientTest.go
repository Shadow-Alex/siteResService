package main

import (
	"context"
	"time"

	"github.com/astaxie/beego"
	"github.com/micro/go-micro"
	"github.com/micro/go-micro/broker"
	"github.com/micro/go-micro/registry"
	"github.com/micro/go-micro/registry/etcd"
	"github.com/micro/go-plugins/broker/nsq"
	"github.com/pborman/uuid"
	log "github.com/sirupsen/logrus"

	cm "../../common"
	pb "../../proto"
)

/********************************************************************
*	 publish four event (pageID) to topic
*    for test service function
 ********************************************************************/

// send events using the publisher
func sendEv(pub micro.Publisher, topic string, msgs []string) {
	for _, msg := range msgs {
		// create new event
		event := &pb.Event{ // make deliver msg as a event which defined at proto file
			Id:        uuid.NewUUID().String(),
			Timestamp: time.Now().Unix(),
			Magic:     -1,
			Message:   msg,
		}

		// publish an event
		if err := pub.Publish(context.Background(), event); err != nil {
			log.WithFields(log.Fields{
				"topic":      topic,
				"event":      event,
				"error info": err.Error(),
			}).Error("[client test]  publishing failed")
		} else {
			log.WithFields(log.Fields{
				"topic": topic,
				"event": event,
			}).Info("[client test]  publishing success")
		}
	}
}

func main() {
	nsqIP := beego.AppConfig.DefaultString("nsq::ip", "localhost")
	nsqPort := beego.AppConfig.DefaultString("nsq::port", "4150")
	var BrokerHosts = []string{
		0: nsqIP + ":" + nsqPort,
	}

	etcdIP := beego.AppConfig.DefaultString("etcd::ip", "localhost")
	etcdPort := beego.AppConfig.DefaultString("etcd::port", "2379")
	var RegistryHosts = []string{
		0: etcdIP + ":" + etcdPort,
	}

	// create a service
	service := micro.NewService(
		micro.Name("go.micro.test.client"),
		micro.Broker(nsq.NewBroker(func(o *broker.Options) {
			o.Addrs = BrokerHosts
		})),
		micro.Registry(etcd.NewRegistry(func(o *registry.Options) { // specified using etcd registry
			o.Addrs = RegistryHosts
		})),
	)

	service.Init()

	// create publisher
	// cm.TopicSUBName means suggestionSeekerServer' subscribe topic,
	// so in this clientTest should publish msg to suggestionSeekerServer's subscribe topic
	topic := beego.AppConfig.DefaultString("nsq::topic.sub", cm.TopicSUBName)
	pub := micro.NewPublisher(topic, service.Client())

	pageIDs := []string{"657349154363294", "1474360329560529", "1554433554795822", "1885990091636729"}

	// pub to topic
	// go sendEv(pub, cm.TOPIC_SUB_NAME, pageID)

	for i := 0; i < 1000000; i++ {
		sendEv(pub, topic, pageIDs)
		time.Sleep(time.Duration(1) * time.Second)
	}

	// block forever
	select {}
}
