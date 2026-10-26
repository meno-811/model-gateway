package main

import (
	"log"
	"net/http"
	"os"

	"morning/internal/gateway"
)

func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func main() {
	pool, err := gateway.NewPool(env("KEYS_FILE", "keys.json"))
	if err != nil {
		log.Fatal(err)
	}
	s := gateway.NewServer(pool, os.Getenv("GATEWAY_API_KEY"), os.Getenv("ADMIN_TOKEN"))
	addr := env("LISTEN_ADDR", ":8080")
	log.Printf("model gateway listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, s.Handler()))
}
