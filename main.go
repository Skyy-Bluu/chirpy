package main

import (
	"database/sql"
	"os"
	"time"

	"github.com/Skyy-Bluu/chirpy/internal/auth"
	"github.com/Skyy-Bluu/chirpy/internal/database"
	"github.com/google/uuid"

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
const chirpID = "chirpID"
const incorrectEmailOrPassword = "Incorrect email or password"
const sixtyDaysDuration = time.Hour * 24 * 60

var profaneWords = []string{
	"kerfuffle", "sharbert", "fornax",
}

type apiConfig struct {
	fileserverHits atomic.Int32
	db             *database.Queries
	platform       string
	secretKey      string
}

type chirp struct {
	Body string `json:"body"`
}

type dbChirp struct {
	ID        string `json:"id"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
	Body      string `json:"body"`
	UserID    string `json:"user_id"`
}

type errorResponse struct {
	Value string `json:"error"`
}

type user struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type dbUser struct {
	ID           string `json:"id"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
	Email        string `json:"email"`
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
}

type refreshTokenResp struct {
	Token string `json:"token"`
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
	user := user{}
	dbUser := dbUser{}
	if err := decoder.Decode(&user); err != nil {
		log.Printf("Error decoding JSON: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	hashedPassword, err := auth.HashPassword(user.Password)

	if err != nil {
		log.Printf("Error hashing password")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	createUserArgs := database.CreateUserParams{
		Email:          user.Email,
		HashedPassword: hashedPassword,
	}

	usr, err := cfg.db.CreateUser(req.Context(), createUserArgs)

	if err != nil {
		log.Printf("Error creating user: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	dbUser.ID = usr.ID.String()
	dbUser.CreatedAt = usr.CreatedAt.String()
	dbUser.UpdatedAt = usr.UpdatedAt.String()
	dbUser.Email = usr.Email

	dat, err := json.Marshal(dbUser)

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

func (cfg *apiConfig) getChirpsHandler(w http.ResponseWriter, req *http.Request) {
	dbChirps, err := cfg.db.GetChirps(req.Context())

	if err != nil {
		log.Printf("Error retrieving chirps: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	chirps := []dbChirp{}

	for _, chirp := range dbChirps {
		chirps = append(chirps, dbChirp{
			ID:        chirp.ID.String(),
			CreatedAt: chirp.CreatedAt.String(),
			UpdatedAt: chirp.UpdatedAt.String(),
			Body:      chirp.Body,
			UserID:    chirp.UserID.String(),
		})
	}

	dat, err := json.Marshal(chirps)

	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(dat)
}

func (cfg *apiConfig) getChirpHandler(w http.ResponseWriter, req *http.Request) {
	id := req.PathValue(chirpID)
	if id == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	chirpUUID, err := uuid.Parse(id)

	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	chirp, err := cfg.db.GetChirp(req.Context(), chirpUUID)

	if err != nil {
		log.Printf("Error retrieving chirp: %s", err)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	dbChirp := dbChirp{
		ID:        chirp.ID.String(),
		CreatedAt: chirp.CreatedAt.String(),
		UpdatedAt: chirp.UpdatedAt.String(),
		Body:      chirp.Body,
		UserID:    chirp.UserID.String(),
	}

	dat, err := json.Marshal(dbChirp)

	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(dat)
}

func (cfg *apiConfig) userLoginHandler(w http.ResponseWriter, req *http.Request) {
	decoder := json.NewDecoder(req.Body)
	user := user{}
	dbUser := dbUser{}
	if err := decoder.Decode(&user); err != nil {
		log.Printf("Error decoding JSON: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	usr, err := cfg.db.GetUserByEmail(req.Context(), user.Email)

	if err != nil {
		log.Printf("Error getting user from DB: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	passwordMatch, err := auth.CheckPasswordHash(user.Password, usr.HashedPassword)

	if err != nil {
		log.Printf("Error checking password hash: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if !passwordMatch {
		w.WriteHeader(http.StatusUnauthorized)
		w.Header().Set("Conten-Type", plainTextContentType)
		io.WriteString(w, incorrectEmailOrPassword)
		return
	}

	jwt, err := auth.MakeJWT(usr.ID, cfg.secretKey, time.Hour)

	if err != nil {
		log.Printf("Error creating JWT: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	refreshToken := auth.MakeRefreshToken()

	createRefreshTokenArgs := database.CreateRefreshTokenParams{
		Token:     refreshToken,
		ExpiresAt: time.Now().Add(sixtyDaysDuration),
		UserID:    usr.ID,
	}

	_, err = cfg.db.CreateRefreshToken(req.Context(), createRefreshTokenArgs)

	if err != nil {
		log.Printf("Error creating refresh token DB entry: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	dbUser.ID = usr.ID.String()
	dbUser.CreatedAt = usr.CreatedAt.String()
	dbUser.UpdatedAt = usr.UpdatedAt.String()
	dbUser.Email = usr.Email
	dbUser.Token = jwt
	dbUser.RefreshToken = refreshToken

	dat, err := json.Marshal(dbUser)

	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", applicationJsonContentType)
	w.WriteHeader(http.StatusOK)
	w.Write(dat)
}

func (cfg *apiConfig) refreshTokenHandler(w http.ResponseWriter, req *http.Request) {
	refreshToken, err := auth.GetBearerToken(req.Header)

	if err != nil {
		log.Printf("Error retrieving  refresh token: %s", err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	tokenParams, err := cfg.db.GetRefreshToken(req.Context(), refreshToken)

	if err != nil {
		log.Printf("Error retrieving  refresh token from DB: %s", err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	timeComaprison := time.Now().Compare(tokenParams.ExpiresAt)

	if tokenParams.RevokedAt.Valid || timeComaprison == +1 || timeComaprison == 0 {
		log.Printf("Expired or revoked refresh token: %s", err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	newToken, err := auth.MakeJWT(tokenParams.UserID, cfg.secretKey, time.Hour)

	if err != nil {
		log.Printf("Error creating JWT: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	refreshTokenResp := refreshTokenResp{
		Token: newToken,
	}

	dat, err := json.Marshal(refreshTokenResp)

	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(dat)
}

func (cfg *apiConfig) revokeRefreshTokenHandler(w http.ResponseWriter, req *http.Request) {
	refreshToken, err := auth.GetBearerToken(req.Header)

	if err != nil {
		log.Printf("Error retrieving  refresh token: %s", err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	if err = cfg.db.RevokeRefreshToken(req.Context(), refreshToken); err != nil {
		log.Printf("Error updating refresh token DB entry: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func healthzHandler(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", plainTextContentType)
	w.WriteHeader(http.StatusOK)
	io.WriteString(w, "OK")
}

func (cfg *apiConfig) chirpHandler(w http.ResponseWriter, req *http.Request) {
	decoder := json.NewDecoder(req.Body)
	w.Header().Set("Content-Type", applicationJsonContentType)

	chirp := chirp{}
	errorResp := errorResponse{}

	if err := decoder.Decode(&chirp); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		errorResp.Value = "Something went wrong"
		dat, err := json.Marshal(errorResp)
		if err != nil {
			log.Printf("Error marshalling JSON: %s", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Write(dat)
		return
	}

	token, err := auth.GetBearerToken(req.Header)

	if err != nil {
		log.Printf("Error retrieving token: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.secretKey)

	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
	}

	if len(chirp.Body) > 140 {
		w.WriteHeader(http.StatusBadRequest)
		errorResp.Value = "Chirp is too long"
		dat, err := json.Marshal(errorResp)
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

	chirpArgs := database.CreateChirpParams{
		Body:   cleanedString,
		UserID: userID,
	}

	createdChirp, err := cfg.db.CreateChirp(req.Context(), chirpArgs)

	if err != nil {
		log.Printf("Internal DB Error creating chirp %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	dbChirp := dbChirp{
		ID:        createdChirp.ID.String(),
		CreatedAt: createdChirp.CreatedAt.String(),
		UpdatedAt: createdChirp.UpdatedAt.String(),
		Body:      createdChirp.Body,
		UserID:    createdChirp.UserID.String(),
	}

	dat, err := json.Marshal(dbChirp)
	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	w.Write(dat)
}

func main() {
	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	platform := os.Getenv("PLATFORM")
	secretKey := os.Getenv("SECRET-KEY")

	db, err := sql.Open("postgres", dbURL)

	if err != nil {
		log.Fatal(err)
	}

	apiConfig := apiConfig{
		fileserverHits: atomic.Int32{},
		db:             database.New(db),
		platform:       platform,
		secretKey:      secretKey,
	}

	mux := http.NewServeMux()
	appHandler := http.StripPrefix("/app", http.FileServer(http.Dir("./")))
	mux.Handle("/app/", apiConfig.middlewareIncrementServerHits(appHandler))
	mux.Handle("/assets/logo.png", http.FileServer(http.Dir(".")))
	mux.HandleFunc("GET /api/healthz", healthzHandler)
	mux.HandleFunc("GET /admin/metrics", apiConfig.showMetricsHandler)
	mux.HandleFunc("POST /admin/reset_metrics", apiConfig.resetHitsMetricHandler)
	mux.HandleFunc("POST /api/chirps", apiConfig.chirpHandler)
	mux.HandleFunc("POST /api/users", apiConfig.createUserHandler)
	mux.HandleFunc("POST /admin/reset", apiConfig.resetDBHandler)
	mux.HandleFunc("GET /api/chirps", apiConfig.getChirpsHandler)
	mux.HandleFunc("GET /api/chirps/{chirpID}", apiConfig.getChirpHandler)
	mux.HandleFunc("POST /api/login", apiConfig.userLoginHandler)
	mux.HandleFunc("POST /api/refresh", apiConfig.refreshTokenHandler)
	mux.HandleFunc("POST /api/revoke", apiConfig.revokeRefreshTokenHandler)

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
