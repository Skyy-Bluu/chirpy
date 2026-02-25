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
	db, err := sql.Open("postgres", dbURL)

	if err != nil {
		log.Fatal(err)
	}

	apiConfig := apiConfig{
		fileserverHits: atomic.Int32{},
		db:             database.New(db),
	}
	mux := http.NewServeMux()
	appHandler := http.StripPrefix("/app", http.FileServer(http.Dir("./")))
	mux.Handle("/app/", apiConfig.middlewareIncrementServerHits(appHandler))
	mux.Handle("/assets/logo.png", http.FileServer(http.Dir(".")))
	mux.HandleFunc("GET /api/healthz", healthzHandler)
	mux.HandleFunc("GET /admin/metrics", apiConfig.showMetricsHandler)
	mux.HandleFunc("POST /admin/reset", apiConfig.resetHitsMetricHandler)
	mux.HandleFunc("POST /api/validate_chirp", validateChirpHandler)

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
