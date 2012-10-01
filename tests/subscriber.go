package main

import (
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"strconv"
//	"time"
//	"math/rand"
)


func publish (count int) {
	for i:=0;i<count;i++{
		url_text := "http://localhost:8080/publish/example"
		resp, err := http.PostForm(url_text, url.Values {"data" : {"This is data on example channel"}})
		fmt.Println("Published to channel" + strconv.Itoa(i), resp, err)
	}
}

func publish_ownchannel (count int) {
	for i:=0;i<count;i++{
		url_text := "http://localhost:8080/publish/own_channel"
		resp, err := http.PostForm(url_text, url.Values {"data" : {"This is data on the self"}})
		fmt.Println("Published to channel" + strconv.Itoa(i), resp, err)
	}
}


var err error
var request_count int = 1
func main () {
	//client := &http.Client{}
	var wg sync.WaitGroup
	wg.Add(request_count)
//	random := rand.New(rand.NewSource(42))

	for i:=0;i<request_count;i++{
	go func (i int) {
		//url := "www.facebook.com"
		url := "http://localhost:8080/subscribe/own_channel"
		resp, err := http.Get(url)
		//	_, err = client.Do(req)
			if err != nil {
				fmt.Println("Error : ", err)
			}
		fmt.Println(resp)
//		subscribe(psc, 1200)
		//publish(1200)
		wg.Done()
	}(i)
	}
	wg.Wait()
	publish(request_count)
	publish_ownchannel(request_count)
}
