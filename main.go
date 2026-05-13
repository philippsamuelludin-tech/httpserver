package main

import (
	"net/http"
	"log"
	"sync/atomic"
	"fmt"
	"encoding/json"
	"strings"
)



func main() {
	apiCfg := apiConfig{}
	serverMux := http.NewServeMux()
	server := &http.Server{
    	Addr:    ":8080",
    	Handler: serverMux,
	}
	serverMux.Handle("/app/", apiCfg.middlewareMetricsInc(http.StripPrefix("/app", http.FileServer(http.Dir(".")))))
	serverMux.HandleFunc("GET /api/healthz", HandlerFunction)
	serverMux.HandleFunc("POST /api/validate_chirp", apiCfg.handlerValidation)
	serverMux.HandleFunc("GET /admin/metrics", apiCfg.handlerMetrics)
	serverMux.HandleFunc("POST /admin/reset", apiCfg.handlerReset)
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
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
    w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf(`<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
  </body>
</html>`, cfg.fileserverHits.Load())))
}

func (cfg *apiConfig) handlerReset(w http.ResponseWriter, r *http.Request) {
    cfg.fileserverHits.Store(0)
}


func (cfg *apiConfig) handlerValidation(w http.ResponseWriter, r *http.Request) {
    type parameters struct {
        Body string `json:"body"`
    }

    decoder := json.NewDecoder(r.Body)
    params := parameters{}
    err := decoder.Decode(&params)
    if err != nil {
		log.Printf("Error decoding parameters: %s", err)
		w.WriteHeader(500)
		return
    }

    type validReturn struct {
        Valid bool `json:"valid"`
    }

	type errorReturn struct {
		Error string `json:"error"`
	}

	type cleanedReturn struct {
		Body string `json:"cleaned_body"`
	}
	
	badWords := map[string]struct{}{
    "kerfuffle": {},
    "sharbert":  {},
    "fornax":    {},
}

	if len(params.Body) > 140 {
		respondWithError(w, 400, "Chirp is too long")
		return
	}

	body := strings.Split(params.Body, " ")
	for i, word := range body {
    if _, ok := badWords[strings.ToLower(word)]; ok {
        body[i] = "****"
    }
}

	params.Body = strings.Join(body, " ")
	respondWithJSON(w, 200, cleanedReturn{Body: params.Body})
    
}

func respondWithJSON(w http.ResponseWriter, code int, payload any) {
    data, err := json.Marshal(payload)
    if err != nil {
        log.Printf("Error marshalling JSON: %s", err)
        w.WriteHeader(500)
        return
    }
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(code)
    w.Write(data)
}

func respondWithError(w http.ResponseWriter, code int, msg string) {
    type errorReturn struct {
        Error string `json:"error"`
    }
    respondWithJSON(w, code, errorReturn{Error: msg})
}