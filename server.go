package main

import (
	"fmt"
	"errors"
	"net/http"
	"regexp"
	"github.com/garyburd/redigo/redis"
	eventsource "github.com/antage/eventsource/http"
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
	//	panic(err)
		return false
	}
	return true
}

func unsubscribe(psc redis.PubSubConn, channel string) bool {
	err := psc.Unsubscribe(channel)
	if err != nil {
	//	panic(err)
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

	verify_exp ,_ := regexp.Compile("/subscribe/[a-zA-Z0-9_]+$")
	channel_exp ,_ := regexp.Compile("/subscribe/")
	if verify_exp.MatchString(r.URL.String()) == true {
		loc := channel_exp.FindIndex([]byte(r.URL.String()))
		channel := r.URL.String()[loc[1]:]
		subscribe_handler(w, r, channel)
	} else {
		fmt.Fprintf(w, "Hi there, I love %s! %s", r.URL.Path[1:], r.URL.Query())
	}
}

func subscribe_handler(w http.ResponseWriter, r *http.Request, request_channel string) {

	psc := redis.PubSubConn{connection}
	channel_es, ok := channel_map[request_channel]
	if ok == false {
		channel_es = eventsource.New()
		channel_map[request_channel] = channel_es
		subscribe(psc, request_channel)
	}

	/*This is a blocking call ! keep this as the end*/
	channel_es.ServeHTTP(w, r)
	if channel_es.ConsumersCount() == 1 {
		go func () {
			for {
				if channel_es.ConsumersCount() == 0 {
					channel_es.Close()
					delete (channel_map, request_channel)
					unsubscribe(psc, request_channel)
					return
				}
			}
		} ()
	}
}

var connection redis.Conn
var channel_map map[string]eventsource.EventSource

func main() {
	var err error
	connection, err = dial()
	if err != nil {
		panic(err)
	}
	defer connection.Close()
	channel_map = make(map[string]eventsource.EventSource)

	psc := redis.PubSubConn{connection}
	// This goroutine receives and prints pushed messages from the server. The
	// goroutine exits when the connection is unsubscribed from all channels or
	// there is an error.
	go reciever(psc)

	http.HandleFunc("/", handler)

	http.Handle("/static/", http.StripPrefix("/static", http.FileServer(http.Dir("/home/jgn438/workspace/helpshift_assignment/go/src/gothroughthis/static"))))
	http.ListenAndServe(":8080", nil)
}
