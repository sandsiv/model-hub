package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
)

func main() {
	wg := &sync.WaitGroup{}

	sendRequest(wg, 100)
	sendRequest(wg, 200)
	sendRequest(wg, 300)
	wg.Wait()
	fmt.Println("All requests sent.")
}

func sendRequest(wg *sync.WaitGroup, priority int) {
	url := "http://127.0.0.1:7766/predict"
	contentType := "application/json"
	requestBody, _ := json.Marshal(map[string]interface{}{
		"parameters": map[string]interface{}{
			"model":    "model2",
			"priority": priority,
		},
	})
	// Launching 100 goroutines for concurrent request sending
	for i := 0; i < 100; i++ {
		go sendPostRequest(wg, url, contentType, requestBody, strconv.Itoa(priority)+"--"+strconv.Itoa(i))
	}
}

func sendPostRequest(wg *sync.WaitGroup, url string, contentType string, body []byte, meta string) {
	wg.Add(1)
	defer wg.Done()
	fmt.Println("Sending request")
	resp, err := http.Post(url, contentType, bytes.NewBuffer(body))
	fmt.Println("Send " + meta)
	if err != nil {
		fmt.Println("Error sending request:", err)
		fmt.Println("Retry")
		sendPostRequest(wg, url, contentType, body, meta)
		return
	}
	defer resp.Body.Close()

	fmt.Println("Request sent, status:", resp.Status)
	return
}
