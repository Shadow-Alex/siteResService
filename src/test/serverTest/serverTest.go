package main

import (
	"context"

	"github.com/astaxie/beego"
	"github.com/micro/go-micro"
	"github.com/micro/go-micro/broker"
	"github.com/micro/go-micro/metadata"
	"github.com/micro/go-micro/registry"
	"github.com/micro/go-micro/registry/etcd"
	"github.com/micro/go-micro/server"
	"github.com/micro/go-plugins/broker/nsq"
	log "github.com/sirupsen/logrus"

	cm "siteResService/src/common"
	pb "siteResService/src/proto"
)

/********************************************************************
*	 subscriber event from topic
*    for test service function
 ********************************************************************/

// Alternatively a function can be used
func process(ctx context.Context, event *pb.Event) error {
	md, _ := metadata.FromContext(ctx)
	log.WithFields(log.Fields{
		"metadata": md,
		"event":    event,
	}).Info("[server test]  Received event")

	return nil
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

	service := micro.NewService(
		micro.Name("go.micro.test.server"),
		micro.Broker(nsq.NewBroker(func(o *broker.Options) { // specified using nsq broker
			o.Addrs = BrokerHosts
		})),
		micro.Registry(etcd.NewRegistry(func(o *registry.Options) { // specified using etcd registry
			o.Addrs = RegistryHosts
		})),
	)

	// micro service init
	service.Init()

	// register subscriber with queue, each message is delivered to a unique subscriber
	// cm.TopicPUBName means suggestionSeekerServer' publish topic,
	//so in this serverTest should subscribe suggestionSeekerServer' publish topic
	topic := beego.AppConfig.DefaultString("nsq::topic.pub", cm.TopicPUBName)
	micro.RegisterSubscriber(topic, service.Server(), process, server.SubscriberQueue("queue.page_id")) // specified a channel name

	if err := service.Run(); err != nil {
		log.Fatal(err)
	}
}
