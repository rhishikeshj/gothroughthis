package main

import (
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"sync"
)

var err error

func main() {

	var run_mode, number_of_channels, number_of_requests int
	var data string
	flag.IntVar(&run_mode, "mode", 0, "0 : Add subscribers 1 : Publish to channels")
	flag.IntVar(&number_of_channels, "num", 0, "How many channels to run command on. Channel names will be \"channel-no\" where no runs from 0 to num")
	flag.IntVar(&number_of_requests, "rnum", 0, "How many requests to execute.")
	flag.StringVar(&data, "data", "", "The data that you want to publish. Ignored for subscribe mode")
	flag.Parse()

	var wait sync.WaitGroup
	wait.Add(number_of_channels)
	if run_mode == 0 {
		for i := 0; i < number_of_channels; i++ {
			go func(i int) {
				ch_url := "http://localhost:8080/subscribe/channel" + strconv.Itoa(i)
				_, err := http.Get(ch_url)
				if err != nil {
					fmt.Println("Error : ", err)
				}
			}(i)
		}
	} else if run_mode == 1 {
		for i := 0; i < number_of_channels; i++ {
			go func(i int) {
				for j := 0; j < number_of_requests; j++ {
					ch_url := "http://localhost:8080/publish/channel" + strconv.Itoa(i)
					_, err := http.PostForm(ch_url, url.Values{"data": {data + " Request number " + strconv.Itoa(j)}})
					if err != nil {
						fmt.Println("Error : ", err)
					}
				}
				wait.Done()
			}(i)
		}
	} else {
		fmt.Println("Fuck off")
	}
	wait.Wait()
}
