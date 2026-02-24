package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"sync/atomic"
)

const plainTextContentType = "text/plain; charset=utf-8"

type apiConfig struct {
	fileserverHits atomic.Int32
}

func (cfg *apiConfig) middlewareIncrementServerHits(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, req *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, req)
	}
	return http.HandlerFunc(fn)
}

func (cfg *apiConfig) showMetricsHandler(w http.ResponseWriter, req *http.Request) {
	w.Header().Add("Content-Type", plainTextContentType)
	w.WriteHeader(http.StatusOK)
	hitsString := fmt.Sprintf("Hits: %d", cfg.fileserverHits.Load())
	io.WriteString(w, hitsString)
}

func (cfg *apiConfig) resetHitsMetricHandler(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(http.StatusOK)
	cfg.fileserverHits.Store(0)
}

func main() {
	apiConfig := apiConfig{
		fileserverHits: atomic.Int32{},
	}
	mux := http.NewServeMux()
	appHandler := http.StripPrefix("/app", http.FileServer(http.Dir("./")))
	mux.Handle("/app/", apiConfig.middlewareIncrementServerHits(appHandler))
	mux.Handle("/assets/logo.png", http.FileServer(http.Dir(".")))
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", plainTextContentType)
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, "OK")
	})
	mux.HandleFunc("/metrics", apiConfig.showMetricsHandler)
	mux.HandleFunc("/reset", apiConfig.resetHitsMetricHandler)

	httpServer := http.Server{
		Handler: mux,
		Addr:    ":8080",
	}

	if err := httpServer.ListenAndServe(); err != nil {
		log.Fatal(err.Error())
	}

}
