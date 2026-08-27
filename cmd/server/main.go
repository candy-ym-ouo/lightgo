package main

import (
	"flag"
	"lightgo/internal/store"
	"lightgo/lightgo"
	"lightgo/lightgo/middlewares"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	defaultPort := os.Getenv("PORT")
	if defaultPort == "" {
		defaultPort = "8080"
	}
	port := flag.String("port", defaultPort, "HTTP server port")
	web := flag.String("web", "web", "path to web assets")
	flag.Parse()
	webDir, err := filepath.Abs(*web)
	if err != nil {
		log.Fatal(err)
	}
	if info, err := os.Stat(webDir); err != nil || !info.IsDir() {
		log.Fatalf("web directory %q is unavailable; run from the project root or pass -web", webDir)
	}
	engine := lightgo.New()
	engine.Use(
		middlewares.RequestID(),
		middlewares.Logger(nil),
		middlewares.Recovery(nil),
		middlewares.CORS(middlewares.CORSConfig{}),
		middlewares.Gzip(-1),
	)
	data := store.New()
	if err := registerRoutes(engine, data, webDir); err != nil {
		log.Fatal(err)
	}
	engine.PrintRoutes()
	addr := *port
	if !strings.HasPrefix(addr, ":") {
		addr = ":" + addr
	}
	if err := engine.Run(addr); err != nil {
		log.Fatal(err)
	}
}
