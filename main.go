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

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"

	"httpserver/internal/database"
	"httpserver/internal/auth"
)

type Chirp struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body      string    `json:"body"`
	UserID    uuid.UUID `json:"user_id"`
}

type User struct {
	ID 		uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email string `json:"email"`
	Token string `json:"token"`
	RefreshToken string `json:"refresh_token"`
}

func main() {
	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	platForm := os.Getenv("PLATFORM")
	sec := os.Getenv("SECRET")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		fmt.Printf("%v", err)
	}
	dbQueries := database.New(db)
	apiCfg := apiConfig{database: dbQueries, platform: platForm, secret: sec}
	log.Printf("JWT secret loaded, length: %d", len(apiCfg.secret))
	serverMux := http.NewServeMux()
	server := &http.Server{
		Addr:    ":8080",
		Handler: serverMux,
	}

	serverMux.Handle("/app/", apiCfg.middlewareMetricsInc(http.StripPrefix("/app", http.FileServer(http.Dir(".")))))
	serverMux.HandleFunc("POST /api/users", apiCfg.handlerUsers)
	serverMux.HandleFunc("POST /api/chirps", apiCfg.handlerChirps)
	serverMux.HandleFunc("POST /api/login", apiCfg.handlerLogin)
	serverMux.HandleFunc("POST /api/refresh", apiCfg.handlerRefresh)
	serverMux.HandleFunc("POST /api/revoke", apiCfg.handlerRevoke)
	serverMux.HandleFunc("GET /api/healthz", HandlerFunction)
	serverMux.HandleFunc("GET /api/chirps", apiCfg.handlerGetChirps)
	serverMux.HandleFunc("GET /api/chirps/{chirpID}", apiCfg.handlerGetChirpsByID)
	serverMux.HandleFunc("GET /admin/metrics", apiCfg.handlerMetrics)
	serverMux.HandleFunc("POST /admin/reset", apiCfg.handlerReset)
	server.ListenAndServe()
}

type apiConfig struct {
	fileserverHits atomic.Int32
	database       *database.Queries
	platform       string
	secret		   string
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) handlerUsers(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Password string `json:"password"`
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

	hashed_password, err := auth.HashPassword(params.Password)
	if err != nil {
		log.Printf("Couldn't hash password: %s", err)
		return
	}

	usr, err := cfg.database.CreateUser(r.Context(), database.CreateUserParams{Email: params.Email, HashedPassword: hashed_password})
	if err != nil {
		fmt.Printf("%v", err)
		w.WriteHeader(500)
		return
	}

	type emailDataReturn struct {
		Id          uuid.UUID `json:"id"`
		CreatedDate time.Time `json:"created_at"`
		UpdateDate  time.Time `json:"updated_at"`
		Email       string    `json:"email"`
	}

	respondWithJSON(w, 201, emailDataReturn{Id: usr.ID, CreatedDate: usr.CreatedAt, UpdateDate: usr.UpdatedAt, Email: usr.Email})

}

func (cfg *apiConfig) handlerChirps(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Body   string    `json:"body"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		log.Printf("Error decoding parameters: %s", err)
		w.WriteHeader(500)
		return
	}

	userjwt, err := auth.GetBearerToken(r.Header)
	if err != nil {
		log.Printf("Error getting user token: %s", err)
		w.WriteHeader(401)
		return
	}
	usrID, err := auth.ValidateJWT(userjwt, cfg.secret)
	if err != nil {
		log.Printf("Error validating user token: %s", err)
		w.WriteHeader(401)
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
	chirp, err := cfg.database.CreateChirp(r.Context(), database.CreateChirpParams{
		Body:   params.Body, // the cleaned version
		UserID: usrID,
	})

	if err != nil {
		log.Printf("CreateChirp error: %s", err)
		respondWithError(w, 500, "Couldn't create chirp")
		return
	}

	respondWithJSON(w, 201, Chirp{
		ID:        chirp.ID,
		CreatedAt: chirp.CreatedAt,
		UpdatedAt: chirp.UpdatedAt,
		Body:      chirp.Body,
		UserID:    chirp.UserID,
	})

}

func (cfg *apiConfig) handlerLogin(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Password   	string    	`json:"password"`
		Email 		string 		`json:"email"`
		ExpiresInSeconds *int 	`json:"expires_in_seconds"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		log.Printf("Error decoding parameters: %s", err)
		w.WriteHeader(500)
		return
	}

	usr, err := cfg.database.LookUpUser(r.Context(), params.Email)
	if err != nil {
		log.Printf("Incorrect email or password: %s", err)
		w.WriteHeader(401)
		return
	}

	valid, err := auth.CheckPasswordHash(params.Password, usr.HashedPassword)
	if err != nil || valid == false {
		log.Printf("Incorrect email or password: %s", err)
		w.WriteHeader(401)
		return
	}

	
	token, err := auth.MakeJWT(usr.ID, cfg.secret, time.Hour)
	if err != nil {
		log.Printf("Error creating Token: %s", err)
		w.WriteHeader(401)
		return
	}

	refreshToken, err := auth.MakeRefreshToken()
	if err != nil {
		log.Printf("Error creating Refreshtoken: %s", err)
		w.WriteHeader(401)
		return
	}
	rfToken, err := cfg.database.InsertTokenToDatabase(r.Context(), database.InsertTokenToDatabaseParams{
		Token: refreshToken,
		UserID: usr.ID,
	})
	if err != nil {
		log.Printf("Error inserting Refreshtoken: %s", err)
		w.WriteHeader(500)
		return
	}


	respondWithJSON(w, 200, User{
		ID: usr.ID,
		CreatedAt: usr.CreatedAt,
		UpdatedAt: usr.UpdatedAt,
		Email: usr.Email,
		Token: token,
		RefreshToken: rfToken.Token,
	})
}

func (cfg *apiConfig) handlerRefresh(w http.ResponseWriter, r *http.Request) {
	bearerToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		log.Printf("Error getting Bearertoken: %s", err)
		w.WriteHeader(401)
		return
	}

	usr, err := cfg.database.GetUserFromRefreshToken(r.Context(), bearerToken)
	if err != nil {
		log.Printf("Error getting User: %s", err)
		w.WriteHeader(401)
		return
	}

	jwtToken, err := auth.MakeJWT(usr.ID, cfg.secret, time.Hour)
	if err != nil {
		log.Printf("Error creating JWT Token: %s", err)
		w.WriteHeader(500)
		return
	}


	type response struct {
		Token string `json:"token"`
	}
	respondWithJSON(w, 200, response{Token: jwtToken})
}

func (cfg *apiConfig) handlerRevoke(w http.ResponseWriter, r *http.Request) {
	bearerToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		log.Printf("Error getting Bearertoken: %s", err)
		w.WriteHeader(401)
		return
	}

	err = cfg.database.RevokeUserRefreshToken(r.Context(), bearerToken)
	if err != nil {
		log.Printf("Error revoking User Refresh Token: %s", err)
		w.WriteHeader(500)
		return
	}

	w.WriteHeader(204)
}

func HandlerFunction(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte("OK"))
}

func (cfg *apiConfig) handlerGetChirps(w http.ResponseWriter, r *http.Request) {
	chirps, err := cfg.database.GetChirps(r.Context())
	if err != nil {
		log.Printf("Error gettting all Chirps: %s", err)
		w.WriteHeader(500)
		return
	}

	chirpSlice := []Chirp{}
	for _, dbChirp := range chirps {
		// convert dbChirp to your Chirp type and append to chirpSlice
		chirpSlice = append(chirpSlice, Chirp{ID: dbChirp.ID, CreatedAt: dbChirp.CreatedAt, UpdatedAt: dbChirp.UpdatedAt, Body: dbChirp.Body, UserID: dbChirp.UserID})
	}
	respondWithJSON(w, 200, chirpSlice)
}

func (cfg *apiConfig) handlerGetChirpsByID(w http.ResponseWriter, r *http.Request) {
	chirpIDStr := r.PathValue("chirpID")
	chirpID, err := uuid.Parse(chirpIDStr)
	if err != nil {
		respondWithError(w, 400, "Invalid chirp ID")
		return
	}

	chirp, err := cfg.database.GetChirpsByID(r.Context(), chirpID)
	if err != nil {
		if err == sql.ErrNoRows {
			respondWithError(w, 404, "Chirp not found")
			return
		}
		log.Printf("Error getting chirp: %s", err)
		respondWithError(w, 500, "Couldn't retrieve chirp")
		return
	}

	respondWithJSON(w, 200, Chirp{
		ID:        chirp.ID,
		CreatedAt: chirp.CreatedAt,
		UpdatedAt: chirp.UpdatedAt,
		Body:      chirp.Body,
		UserID:    chirp.UserID,
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
