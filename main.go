package main

import (
	"io"
	"log"
	"net/http"
)

func main() {
	mux := http.NewServeMux()
	mux.Handle("/app/", http.StripPrefix("/app", http.FileServer(http.Dir("./"))))
	mux.Handle("/assets/logo.png", http.FileServer(http.Dir(".")))
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, "OK")
	})

	httpServer := http.Server{
		Handler: mux,
		Addr:    ":8080",
	}

	if err := httpServer.ListenAndServe(); err != nil {
		log.Fatal(err.Error())
	}

}
