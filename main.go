package main

import (
	"net/http"
	"log"
	"sync/atomic"
	"fmt"
)

func main() {
	apiCfg := apiConfig{}
	serverMux := http.NewServeMux()
	server := &http.Server{
    	Addr:    ":8080",
    	Handler: serverMux,
	}
	serverMux.Handle("/app/", apiCfg.middlewareMetricsInc(http.StripPrefix("/app", http.FileServer(http.Dir(".")))))
	serverMux.HandleFunc("GET /healthz", HandlerFunction)
	serverMux.HandleFunc("GET /metrics", apiCfg.handlerMetrics)
	serverMux.HandleFunc("POST /reset", apiCfg.handlerReset)
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



type apiConfig struct {
	fileserverHits atomic.Int32
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        cfg.fileserverHits.Add(1)
        next.ServeHTTP(w, r)
    })
}

func (cfg *apiConfig) handlerMetrics(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintf(w, "Hits: %d", cfg.fileserverHits.Load())
}

func (cfg *apiConfig) handlerReset(w http.ResponseWriter, r *http.Request) {
    cfg.fileserverHits.Store(0)
}


