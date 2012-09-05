package main

import (
	"fmt"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"github.com/garyburd/redigo/redis"
	eventsource "github.com/rhishikeshj/eventsource/http"
)

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

func unsubscribe(psc redis.PubSubConn, channel string) bool {
	err := psc.Unsubscribe(channel)
	if err != nil {
		panic(err)
		return false
	}
	return true
}

func reciever (psc redis.PubSubConn) {
	for {
		switch n := psc.Receive().(type) {
			case redis.Message:
				fmt.Printf("%s: message: %s\n", n.Channel, n.Data)
				channel_es,ok := channel_map[n.Channel]
				if ok == false {
					// error
					return
				} else {
					channel_es.SendMessage(string(n.Data), "","")
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

func handler(w http.ResponseWriter, r *http.Request) {

	verify_exp ,_ := regexp.Compile("/subscribe/([a-zA-Z0-9_]+)(/[a-zA-Z0-9_]+)*$")
	if verify_exp.MatchString(r.URL.String()) == true {
		channel_list := r.URL.String()[11:]
		channels := strings.Split(channel_list, "/")
		subscribe_handler(w, r, channels)
	} else {
		fmt.Fprintf(w, "Hi there, Try adding a subscription by doing a GET to /subscribe/<channel-name>")
	}
}

func cleanup_connection (channel_es eventsource.EventSource, request_channel string) {
	channel_es.Close()
	delete (channel_map, request_channel)
	unsubscribe(psc_map[request_channel], request_channel)
	delete (psc_map, request_channel)
}

func subscribe_handler(w http.ResponseWriter, r *http.Request, request_channels []string) {

	connection := eventsource.GetConnection(w, r)
	for _, request_channel := range request_channels {
		map_mutex.Lock()
		channel_es, ok := channel_map[request_channel]
		if ok == false {
			psc := redis.PubSubConn{redis_connection}
			go reciever(psc)
			channel_es = eventsource.New()
			channel_map[request_channel] = channel_es
			psc_map[request_channel] = psc
			map_mutex.Unlock()
			subscribe(psc, request_channel)
		}
		channel_es.AddConsumer(connection)
	}

	/*This is a blocking call ! keep this as the end*/
	eventsource.ServeHTTP(connection)
	for _, request_channel := range request_channels {
		channel_es,_ := channel_map[request_channel]
		if channel_es.ConsumersCount() == 1 {
			channel_es.RemoveConsumer(connection)
			map_mutex.Lock()
			cleanup_connection(channel_es, request_channel)
			map_mutex.Unlock()
		} else {
			channel_es.RemoveConsumer(connection)
		}
	}
}

var redis_connection redis.Conn
var channel_map map[string]eventsource.EventSource
var psc_map map[string]redis.PubSubConn
var map_mutex sync.Mutex

func main() {
	var err error
	redis_connection, err = dial()
	if err != nil {
		panic(err)
	}
	defer redis_connection.Close()
	channel_map = make(map[string]eventsource.EventSource)
	psc_map = make(map[string]redis.PubSubConn)

	// This goroutine receives and prints pushed messages from the server. The
	// goroutine exits when the connection is unsubscribed from all channels or
	// there is an error.

	http.HandleFunc("/", handler)

	http.Handle("/static/", http.StripPrefix("/static", http.FileServer(http.Dir("/home/jgn438/workspace/helpshift_assignment/go/src/gothroughthis/static"))))
	http.ListenAndServe(":8080", nil)
}
