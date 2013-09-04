package main

import (
	"errors"
	"fmt"
	"github.com/garyburd/redigo/redis"
	eventsource "github.com/rhishikeshj/eventsource/http"
	"gokaf"
	"net/http"
	"strings"
	"sync"
)

type ServerChannel struct {
	es   eventsource.EventSource
	name string
	pipe chan (string)
}

type PublishData struct {
	channel_name string
	data         string
}

func dial() (redis.Conn, error) {
	c, err := redis.Dial("tcp", ":6379")
	if err != nil {
		return nil, err
	}

	_, err = c.Do("SELECT", "9")
	if err != nil {
		return nil, err
	}

	n, err := redis.Int(c.Do("DBSIZE"))
	if err != nil {
		return nil, err
	}

	if n != 0 {
		return nil, errors.New("Database #9 is not empty, test can not continue")
	}
	return c, nil
}

func subscribe(psc redis.PubSubConn, channel string) bool {
	err := psc.Subscribe(channel)
	if err != nil {
		panic(err)
		return false
	}
	return true
}

func publish(redis_connection redis.Conn, channel, data string) bool {
	redis_connection.Do("PUBLISH", channel, data)
	return true
}

func unsubscribe(psc redis.PubSubConn, channel string) bool {
	err := psc.Unsubscribe(channel)
	if err != nil {
		panic(err)
		return false
	}
	return true
}

func reciever(psc redis.PubSubConn, channel_map map[string]ServerChannel) {
	for {
		fmt.Println("Tick")
		switch n := psc.Receive().(type) {
		case redis.Message:
			server_channel, ok := channel_map[n.Channel]
			if ok == false {
				// error
				return
			} else {
				server_channel.pipe <- string(n.Data)
			}
		case redis.Subscription:
			fmt.Printf("%s: %s %d\n", n.Channel, n.Kind, n.Count)
			if n.Count == 0 {
				return
			}
		case error:
			fmt.Printf("error: %v\n", n)
			return
		}
	}
}

func kafka_receiver(kafka_connection *gokaf.Client) {

}
func removeDuplicates(a []string) []string {
	result := []string{}
	seen := map[string]int{}
	for _, val := range a {
		if _, ok := seen[val]; !ok {
			result = append(result, val)
			seen[val] = 1
		}
	}
	return result
}

func cleanup_connection(channel_es eventsource.EventSource, request_channel string, channel_map map[string]ServerChannel) {
	channel_es.Close()
	delete(channel_map, request_channel)
}

func publish_handler(
	w http.ResponseWriter,
	r *http.Request,
	request_channels []string,
	data string,
	redis_connection redis.Conn,
	publishers_channel chan (PublishData)) {

	for _, request_channel := range request_channels {
		publishers_channel <- PublishData{request_channel, data}
	}
}

func subscribe_handler(
	w http.ResponseWriter,
	r *http.Request,
	request_channels []string,
	channel_map map[string]ServerChannel,
	redis_connection redis.Conn,
	subscribers_channel chan (string)) {

	var map_mutex sync.Mutex
	connection := eventsource.GetConnection(w, r)
	for _, request_channel := range request_channels {
		map_mutex.Lock()
		server_channel, ok := channel_map[request_channel]
		if ok == false {
			server_channel = ServerChannel{}
			server_channel.name = request_channel
			server_channel.es = eventsource.New()
			server_channel.pipe = make(chan string)
			go func(server_channel ServerChannel) {
				for {
					data := <-server_channel.pipe
					fmt.Println("Got : ", data, " on channel ", server_channel.name)
					server_channel.es.SendMessage(data, "", "")
				}
			}(server_channel)
			channel_map[request_channel] = server_channel
			subscribers_channel <- request_channel
		}
		map_mutex.Unlock()
		server_channel.es.AddConsumer(connection)
	}

	/*This is a blocking call ! keep this as the end*/
	eventsource.ServeHTTP(connection)
	for _, request_channel := range request_channels {
		server_channel, _ := channel_map[request_channel]
		if server_channel.es.ConsumersCount() == 1 {
			server_channel.es.RemoveConsumer(connection)
			map_mutex.Lock()
			cleanup_connection(server_channel.es, request_channel, channel_map)
			map_mutex.Unlock()
		} else {
			server_channel.es.RemoveConsumer(connection)
		}
	}
}

func main() {

	redis_p_connection, err := dial()
	if err != nil {
		panic(err)
	}
	defer redis_p_connection.Close()

	redis_s_connection, err := dial()
	if err != nil {
		panic(err)
	}
	defer redis_s_connection.Close()
	subscribers_channel := make(chan string)
	publishers_channel := make(chan PublishData)

	channel_map := make(map[string]ServerChannel)
	redis_subscriber := redis.PubSubConn{redis_s_connection}
	go func(psc redis.PubSubConn) {
		for {
			select {
			case s_channel := <-subscribers_channel:
				subscribe(psc, s_channel)
			case publish_data := <-publishers_channel:
				publish(redis_p_connection, publish_data.channel_name, publish_data.data)
			}
		}
	}(redis_subscriber)

	go reciever(redis_subscriber, channel_map)

	/*
		Kafka related code
	*/

	kafka_connection, err := gokaf.Connect(1, []string{"127.0.0.1:9092"})
	topics, _ := kafka_connection.Topics()
	for _, topic := range topics {
		go func() {
			kaf_consumer := gokaf.Consume(kafka_connection, topic, "group1", 0)
			defer kaf_consumer.Consumer.Close()
			for {
				data := <-kaf_consumer.Channel
				fmt.Println("The data on the channel ", topic, "is : ", string(data.Value), " and the offset is ", data.Offset)
			}
		}()
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		components := strings.Split(r.URL.Path[1:], "/")
		if components[0] == "subscribe" {
			channels := removeDuplicates(components[1:])
			subscribe_handler(w, r, channels, channel_map, redis_s_connection, subscribers_channel)
		} else if components[0] == "publish" {
			channels := removeDuplicates(components[1:])
			publish_handler(w, r, channels, r.FormValue("data"), redis_p_connection, publishers_channel)
		} else {
			fmt.Println("Fuck off")
		}
	})
	http.Handle("/static/", http.StripPrefix("/static", http.FileServer(http.Dir("/home/rhishikeshjoshi/workspace/goPlay/gothroughthis/static"))))
	http.ListenAndServe(":8080", nil)
}
