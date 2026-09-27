package main

import (
	"context"
	"github.com/joho/godotenv"
	"log"
	"net/http"
	"os"
	"termorize/src/classification"
	"time"
)

func main() {
	_ = godotenv.Load()
	secret := os.Getenv("SECRET")
	if secret == "" {
		log.Fatal("SECRET is required")
	}
	backendURL := os.Getenv("BACKEND_INTERNAL_URL")
	if backendURL == "" {
		backendURL = "http://backend:8080"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := classification.Trigger(ctx, &http.Client{Timeout: 10 * time.Second}, backendURL, secret); err != nil {
		log.Fatal(err)
	}
	log.Print("classification sweep requested")
}
