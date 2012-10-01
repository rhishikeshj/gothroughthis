package main

import (
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"errors"
	"strconv"
//	"time"
        "github.com/garyburd/redigo/redis"
//	"math/rand"
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



func subscribe(psc redis.PubSubConn, count int) bool {
	for i:=0;i<count;i++ {
		err := psc.Subscribe("channel" + strconv.Itoa(i))
		fmt.Println("Subscribed to ", "channel" + strconv.Itoa(i))
	        if err != nil {
			panic(err)
			return false
		}
	}
        return true
}

func publish (count int) {
	for i:=0;i<count;i++{
		url_text := "http://localhost:8080/publish/example"
		resp, err := http.PostForm(url_text, url.Values {"data" : {"This is data on example channel"}})
		//redis_connection.Do("PUBLISH", "channel" + strconv.Itoa(i), "This is data on channel" + strconv.Itoa(i))
		fmt.Println("Published to channel" + strconv.Itoa(i), resp, err)
	}
}

func publish_ownchannel (count int) {
	for i:=0;i<count;i++{
		url_text := "http://localhost:8080/publish/own_channel"
		resp, err := http.PostForm(url_text, url.Values {"data" : {"This is data on the self"}})
		//redis_connection.Do("PUBLISH", "channel" + strconv.Itoa(i), "This is data on channel" + strconv.Itoa(i))
		fmt.Println("Published to channel" + strconv.Itoa(i), resp, err)
	}
}


var err error
var redis_connection redis.Conn
var request_count int = 1
func main () {
	//client := &http.Client{}
	var wg sync.WaitGroup
	wg.Add(request_count)
//	random := rand.New(rand.NewSource(42))
        redis_connection, err = dial()
        if err != nil {
                panic(err)
        }
        defer redis_connection.Close()
	//psc := redis.PubSubConn{redis_connection}

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
