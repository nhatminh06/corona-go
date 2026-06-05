package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	version   = getenv("APP_VERSION", "dev")
	startTime = time.Now()
)

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// readSecretFile reads key=value pairs from the Vault Agent secret file
func readSecretFile(path string) map[string]string {
	secrets := make(map[string]string)
	f, err := os.Open(path)
	if err != nil {
		return secrets
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			secrets[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
	return secrets
}

func helloHandler(w http.ResponseWriter, r *http.Request) {
	secrets := readSecretFile("/vault/secrets/app-creds")

	resp := map[string]any{
		"service":   "corona-go",
		"requestID": uuid.New().String(),
		"message":   "hello from the Go service",
		"secretsLoaded": map[string]bool{
			"apiKey":        secrets["apiKey"] != "",
			"externalToken": secrets["externalServiceToken"] != "",
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
