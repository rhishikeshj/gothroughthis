package main

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"
	eventsource "github.com/rhishikeshj/eventsource/http"
)


func reciever (data_channel_name string) {
	fmt.Println("Adding a receiver for " + data_channel_name)
	data_channel := channel_map[data_channel_name]
	channel_es := event_source_map[data_channel_name]
	for {
		data := <-data_channel
		channel_es.SendMessage(data, "", "")
		fmt.Println("Got this as data", data, "for channel : " + data_channel_name)
	}
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


func handler(w http.ResponseWriter, r *http.Request) {

	subscribe_exp ,_ := regexp.Compile("/subscribe/([a-zA-Z0-9_]+)(/[a-zA-Z0-9_]+)*$")
	publish_exp ,_ := regexp.Compile("/publish/([a-zA-Z0-9_]+)(/[a-zA-Z0-9_]+)*$")
	if subscribe_exp.MatchString(r.URL.String()) == true {
		channel_list := r.URL.String()[11:]
		channels := strings.Split(channel_list, "/")
		channels = removeDuplicates(channels)
		subscribe_handler(w, r, channels)
	} else if publish_exp.MatchString(r.URL.String()) == true {
		fmt.Println(r.URL.String())
		channel_list := r.URL.String()[9:]
		channels := strings.Split(channel_list, "/")
		channels = removeDuplicates(channels)
		pub_channel := channels[0]
		fmt.Println(pub_channel)
		data_channel := channel_map[pub_channel]
		data_channel <- r.FormValue("data")
	} else {
		fmt.Println(w, "Hi there, Try adding a subscription by doing a GET to /subscribe/<channel-name>")
	}
}

func subscribe_handler(w http.ResponseWriter, r *http.Request, request_channels []string) {

	connection := eventsource.GetConnection(w, r)
	for _, request_channel := range request_channels {
		map_mutex.Lock()
		_, ok := channel_map[request_channel]

		if ok == false {
			data_channel := make(chan string)
			channel_map[request_channel] = data_channel
			event_source_map [request_channel] = eventsource.New()
			fmt.Println("Creating a data channel for " + request_channel, data_channel, channel_map[request_channel])
			go reciever(request_channel)
		}
		event_source_map [request_channel].AddConsumer(connection)
		map_mutex.Unlock()
	}

	/*This is a blocking call ! keep this as the end*/
	eventsource.ServeHTTP(connection)
}

var channel_map map[string](chan string)
var event_source_map map[string]eventsource.EventSource

var map_mutex, receive_mutex sync.Mutex

func main() {
	channel_map = make(map[string](chan string))
        event_source_map = make(map[string]eventsource.EventSource)


	// This goroutine receives and prints pushed messages from the server. The
	// goroutine exits when the connection is unsubscribed from all channels or
	// there is an error.

	http.HandleFunc("/", handler)

	http.Handle("/static/", http.StripPrefix("/static", http.FileServer(http.Dir("/home/jgn438/workspace/helpshift_assignment/go/src/gothroughthis/static"))))
	http.ListenAndServe(":8080", nil)
}
