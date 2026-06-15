package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/go-redis/redis/v8"
)

var (
	version   = getenv("APP_VERSION", "dev")
	startTime = time.Now()

	eventsMu sync.Mutex
	events   []string

	rdb *redis.Client
)

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

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

func startKafkaConsumer() {
	broker := getenv("KAFKA_BROKER", "kafka-external.messaging.svc.cluster.local:9092")
	topic := getenv("KAFKA_TOPIC", "corona-events")

	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  []string{broker},
		GroupID:  "corona-go-group",
		Topic:    topic,
		MinBytes: 1,
		MaxBytes: 10e6,
	})
	defer r.Close()

	log.Printf("Kafka consumer started: broker=%s topic=%s", broker, topic)

	for {
		m, err := r.ReadMessage(context.Background())
		if err != nil {
			log.Printf("Kafka read error: %v — retrying in 5s", err)
			time.Sleep(5 * time.Second)
			continue
		}
		msg := fmt.Sprintf("[%s] %s", time.Now().Format(time.RFC3339), string(m.Value))
		log.Printf("Kafka event received: %s", msg)

		eventsMu.Lock()
		events = append(events, msg)
		if len(events) > 100 {
			events = events[len(events)-100:]
		}
		eventsMu.Unlock()

		if rdb != nil {
			rdb.Incr(context.Background(), "corona-go:events:count")
		}
	}
}

func initRedis() {
	addr := getenv("REDIS_ADDR", "redis.messaging.svc.cluster.local:6379")
	rdb = redis.NewClient(&redis.Options{Addr: addr})
	_, err := rdb.Ping(context.Background()).Result()
	if err != nil {
		log.Printf("Redis not available: %v", err)
		rdb = nil
	} else {
		log.Printf("Redis connected: %s", addr)
	}
}

func helloHandler(w http.ResponseWriter, r *http.Request) {
	secrets := readSecretFile("/vault/secrets/app-creds")
	resp := map[string]any{
		"service":   "corona-go",
    		"version": "v1",
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

func eventsHandler(w http.ResponseWriter, r *http.Request) {
	eventsMu.Lock()
	defer eventsMu.Unlock()

	var redisCount string
	if rdb != nil {
		val, err := rdb.Get(context.Background(), "corona-go:events:count").Result()
		if err == nil {
			redisCount = val
		}
	}

	resp := map[string]any{
		"service":     "corona-go",
		"eventCount":  len(events),
		"redisCount":  redisCount,
		"recentEvents": events,
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func main() {
	initRedis()
	go startKafkaConsumer()

	http.HandleFunc("/", helloHandler)
	http.HandleFunc("/version", versionHandler)
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/events", eventsHandler)

	port := getenv("PORT", "8080")
	log.Printf("corona-go listening on :%s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
