package main

import (
	"database/sql"
	"os"

	database "github.com/Skyy-Bluu/chirpy/internal/database"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"

	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync/atomic"
)

const plainTextContentType = "text/plain; charset=utf-8"
const htmlTextContentType = "text/html"
const applicationJsonContentType = "application/json"

var profaneWords = []string{
	"kerfuffle", "sharbert", "fornax",
}

type apiConfig struct {
	fileserverHits atomic.Int32
	db             *database.Queries
	platform       string
}

type chirp struct {
	Body string `json:"body"`
}

type errorResponse struct {
	Value string `json:"error"`
}

type cleanedChirp struct {
	CleanedBody string `json:"cleaned_body"`
}

type email struct {
	Value string `json:"email"`
}

type user struct {
	ID        string `json:"id"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
	Email     string `json:"email"`
}

func (cfg *apiConfig) middlewareIncrementServerHits(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, req *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, req)
	}
	return http.HandlerFunc(fn)
}

func (cfg *apiConfig) showMetricsHandler(w http.ResponseWriter, req *http.Request) {
	w.Header().Add("Content-Type", htmlTextContentType)
	w.WriteHeader(http.StatusOK)
	hitsString := fmt.Sprintf(`
<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
  </body>
</html>
`, cfg.fileserverHits.Load())
	io.WriteString(w, hitsString)
}

func (cfg *apiConfig) resetHitsMetricHandler(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(http.StatusOK)
	cfg.fileserverHits.Store(0)
}

func (cfg *apiConfig) createUserHandler(w http.ResponseWriter, req *http.Request) {
	decoder := json.NewDecoder(req.Body)
	email := email{}
	emailResp := user{}
	if err := decoder.Decode(&email); err != nil {
		log.Printf("Error decoding JSON: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	usr, err := cfg.db.CreateUser(req.Context(), email.Value)

	emailResp.ID = usr.ID.String()
	emailResp.CreatedAt = usr.CreatedAt.String()
	emailResp.UpdatedAt = usr.UpdatedAt.String()
	emailResp.Email = usr.Email

	if err != nil {
		log.Printf("Error creating user: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	dat, err := json.Marshal(emailResp)

	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", applicationJsonContentType)
	w.WriteHeader(http.StatusCreated)
	w.Write(dat)
}

func (cfg *apiConfig) resetDBHandler(w http.ResponseWriter, req *http.Request) {
	if cfg.platform != "dev" {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	if err := cfg.db.DeleteUsers(req.Context()); err != nil {
		log.Printf("Error deleting users: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func healthzHandler(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", plainTextContentType)
	w.WriteHeader(http.StatusOK)
	io.WriteString(w, "OK")
}

func validateChirpHandler(w http.ResponseWriter, req *http.Request) {
	decoder := json.NewDecoder(req.Body)
	w.Header().Set("Content-Type", applicationJsonContentType)

	chirp := chirp{}
	error := errorResponse{}
	cleanedChirp := cleanedChirp{}

	if err := decoder.Decode(&chirp); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		error.Value = "Something went wrong"
		dat, err := json.Marshal(error)
		if err != nil {
			log.Printf("Error marshalling JSON: %s", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Write(dat)
		return
	}

	if len(chirp.Body) > 140 {
		w.WriteHeader(http.StatusBadRequest)
		error.Value = "Chirp is too long"
		dat, err := json.Marshal(error)
		if err != nil {
			log.Printf("Error marshalling JSON: %s", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Write(dat)
		return
	}

	chirpSlice := strings.Split(chirp.Body, " ")

	for i, word := range chirpSlice {
		for _, profaneWord := range profaneWords {
			if strings.ToLower(word) == profaneWord {
				chirpSlice[i] = "****"
			}
		}
	}

	cleanedString := strings.Join(chirpSlice, " ")
	w.WriteHeader(http.StatusOK)
	cleanedChirp.CleanedBody = cleanedString
	dat, err := json.Marshal(cleanedChirp)
	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Write(dat)
}

func main() {
	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	platform := os.Getenv("PLATFORM")
	db, err := sql.Open("postgres", dbURL)

	if err != nil {
		log.Fatal(err)
	}

	apiConfig := apiConfig{
		fileserverHits: atomic.Int32{},
		db:             database.New(db),
		platform:       platform,
	}

	mux := http.NewServeMux()
	appHandler := http.StripPrefix("/app", http.FileServer(http.Dir("./")))
	mux.Handle("/app/", apiConfig.middlewareIncrementServerHits(appHandler))
	mux.Handle("/assets/logo.png", http.FileServer(http.Dir(".")))
	mux.HandleFunc("GET /api/healthz", healthzHandler)
	mux.HandleFunc("GET /admin/metrics", apiConfig.showMetricsHandler)
	mux.HandleFunc("POST /admin/reset_metrics", apiConfig.resetHitsMetricHandler)
	mux.HandleFunc("POST /api/validate_chirp", validateChirpHandler)
	mux.HandleFunc("POST /api/users", apiConfig.createUserHandler)
	mux.HandleFunc("POST /admin/reset", apiConfig.resetDBHandler)

	httpServer := http.Server{
		Handler: mux,
		Addr:    ":8080",
	}

	if err := httpServer.ListenAndServe(); err != nil {
		log.Fatal(err.Error())
	}

}

func containsAnyInSlice(s string, slice []string) bool {
	for _, word := range slice {
		if strings.Contains(s, word) {
			return true
		}
	}
	return false
}
