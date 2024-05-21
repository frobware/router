package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {})

	port := 1936
	log.Printf("Server starting on port %d", port)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("SIGTERM received, shutting down...")
		os.Exit(0)
	}()

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}
