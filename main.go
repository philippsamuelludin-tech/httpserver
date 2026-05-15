package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/joho/godotenv"
	"github.com/google/uuid"
	_ "github.com/lib/pq"

	"httpserver/internal/database"
)

type Chirp struct {
		Id uuid.UUID `json:"id"`
		CreatedAt time.Time
		UpdatedAt time.Time
		Body string
		UserId uuid.UUID
	}

func main() {
	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	platForm := os.Getenv("PLATFORM")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		fmt.Printf("%v", err)
	}
	dbQueries := database.New(db)
	apiCfg := apiConfig{database: dbQueries, platform: platForm}
	serverMux := http.NewServeMux()
	server := &http.Server{
		Addr:    ":8080",
		Handler: serverMux,
	}

	serverMux.Handle("/app/", apiCfg.middlewareMetricsInc(http.StripPrefix("/app", http.FileServer(http.Dir(".")))))
	serverMux.HandleFunc("POST /api/users", apiCfg.handlerUsers)
	serverMux.HandleFunc("POST /api/chirps", apiCfg.handlerChirps)
	serverMux.HandleFunc("GET /api/healthz", HandlerFunction)
	serverMux.HandleFunc("GET /admin/metrics", apiCfg.handlerMetrics)
	serverMux.HandleFunc("POST /admin/reset", apiCfg.handlerReset)
	server.ListenAndServe()
}



type apiConfig struct {
	fileserverHits atomic.Int32
	database *database.Queries
	platform string
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) handlerUsers(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Email string `json:"email"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		log.Printf("Error decoding parameters: %s", err)
		w.WriteHeader(500)
		return
	}

	usr, err := cfg.database.CreateUser(r.Context(), params.Email)
	if err != nil {
		fmt.Printf("%v", err)
		w.WriteHeader(500)
		return
	}

	type emailDataReturn struct {
		Id uuid.UUID `json:"id"`
		CreatedDate time.Time `json:"created_at"`
		UpdateDate time.Time `json:"updated_at"`
		Email string `json:"email"`
	}

	respondWithJSON(w, 201, emailDataReturn{Id: usr.ID, CreatedDate: usr.CreatedAt, UpdateDate: usr.UpdatedAt, Email: usr.Email})

}

func (cfg *apiConfig) handlerChirps(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Body string `json:"body"`
		UserID uuid.UUID
	}

	type Chirp struct {
		Id uuid.UUID
		Created_at time.Time
		Updated_at time.Time
		Body string
		User_id uuid.UUID
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		log.Printf("Error decoding parameters: %s", err)
		w.WriteHeader(500)
		return
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
	chirp, err := cfg.db.CreateChirp(r.Context(), database.CreateChirpParams{
		Body:   params.Body, // the cleaned version
		UserID: params.UserID,
	})
	respondWithJSON(w, 200, cleanedReturn{Body: params.Body})

}

func HandlerFunction(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte("OK"))
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
	if cfg.platform != "dev" {
		w.WriteHeader(403)
		return
	}
	err := cfg.database.DeleteUsers(r.Context())
	if err != nil {
		log.Printf("Error deleting database: %s", err)
		w.WriteHeader(500)
		return
	}
	w.WriteHeader(200)
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
