package main

import (
	"net/http"
	"log"
)

func main() {
	serverMux := http.NewServeMux()
	server := &http.Server{
    	Addr:    ":8080",
    	Handler: serverMux,
	}
	serverMux.Handle("/", http.FileServer(http.Dir("")))
	err := server.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}

