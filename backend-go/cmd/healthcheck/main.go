package main

import (
	"net/http"
	"os"
	"time"
)

func main() {
	client := http.Client{Timeout: 3 * time.Second}
	response, err := client.Get("http://127.0.0.1:8080/ready")
	if err != nil || response.StatusCode != http.StatusOK {
		os.Exit(1)
	}
	_ = response.Body.Close()
}
