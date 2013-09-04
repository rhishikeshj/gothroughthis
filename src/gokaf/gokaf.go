package gokaf

import (
	"github.com/Shopify/sarama"
	"time"
)

type Client sarama.Client

type ConsumerChannel struct {
	Consumer *sarama.Consumer
	Channel <-chan *sarama.ConsumerEvent
}

func Connect(id int, addrs []string) (kaf_client *sarama.Client, err error) {
	kaf_client, err = sarama.NewClient("kafka_client_connection_"+"id", addrs, &sarama.ClientConfig{5, 5 * time.Second})
	return
}

func Consume (client *sarama.Client, topic, group_name string, partition_id int32) (kafConsumer ConsumerChannel){
	consumer, err := sarama.NewConsumer(client, topic, partition_id, group_name, nil)
	var data_channel <-chan *sarama.ConsumerEvent
	if err != nil {
    	panic(err)
	} else {
    	data_channel = consumer.Events()
	}
	return ConsumerChannel{consumer, data_channel}
}
