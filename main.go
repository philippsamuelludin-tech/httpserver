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
	serverMux.Handle("/app/", http.StripPrefix("/app", http.FileServer(http.Dir(""))))
	serverMux.HandleFunc("/healthz", HandlerFunction)
	err := server.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}

func HandlerFunction(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte("OK"))
}

