package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

func main() {
	wg := &sync.WaitGroup{}

	go sendRequest(wg, 100)
	go sendRequest(wg, 200)
	go sendRequest(wg, 300)
	time.Sleep(1 * time.Millisecond)
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
		wg.Add(1)
		go sendPostRequest(wg, url, contentType, requestBody)
	}
}

func sendPostRequest(wg *sync.WaitGroup, url string, contentType string, body []byte) {
	defer wg.Done()
	fmt.Println("Sending request")
	resp, err := http.Post(url, contentType, bytes.NewBuffer(body))
	if err != nil {
		fmt.Println("Error sending request:", err)
		return
	}
	defer resp.Body.Close()

	fmt.Println("Request sent, status:", resp.Status)
	return
}
