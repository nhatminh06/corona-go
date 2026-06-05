package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
)

var (
    version         = getenv("APP_VERSION", "dev")
    apiKey          = getenv("APP_API_KEY", "<not-set>")
    externalToken   = getenv("APP_EXTERNAL_TOKEN", "<not-set>")
    startTime       = time.Now()
)

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func helloHandler(w http.ResponseWriter, r *http.Request) {
    resp := map[string]any{
        "service":   "corona-go",
        "requestID": uuid.New().String(),
        "message":   "hello from the Go service",
        "secretsLoaded": map[string]bool{
            "apiKey":        apiKey != "<not-set>",
            "externalToken": externalToken != "<not-set>",
        },
    }
    w.Header().Set("Content-Type", "application/json")
    _ = json.NewEncoder(w).Encode(resp)
}

func versionHandler(w http.ResponseWriter, r *http.Request) {
	resp := map[string]any{
		"version":   version,
		"uptimeSec": int(time.Since(startTime).Seconds()),
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "ok")
}

func main() {
	http.HandleFunc("/", helloHandler)
	http.HandleFunc("/version", versionHandler)
	http.HandleFunc("/health", healthHandler)

	port := getenv("PORT", "8080")
	log.Printf("corona-go listening on :%s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
