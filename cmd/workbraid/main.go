package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"

	"workbraid/internal/web"
)

func main() {
	listenAddress := flag.String("listen", "127.0.0.1:8080", "loopback address for the local WorkBraid server")
	dataDirectory := flag.String("data-dir", defaultDataDirectory(), "WorkBraid application-data directory")
	uiDirectory := flag.String("ui-dir", "frontend/dist", "directory containing the built browser UI")
	flag.Parse()

	expectedOrigin, err := originForLoopbackAddress(*listenAddress)
	if err != nil {
		log.Fatal(err)
	}
	if err := os.MkdirAll(*dataDirectory, 0o700); err != nil {
		log.Fatalf("create application-data directory: %v", err)
	}

	handler := web.NewHandler(expectedOrigin, *uiDirectory, *dataDirectory)
	log.Printf("WorkBraid is available at %s", expectedOrigin)
	if err := http.ListenAndServe(*listenAddress, handler); err != nil {
		log.Fatal(err)
	}
}

func defaultDataDirectory() string {
	if configured := os.Getenv("WORKBRAID_DATA_DIR"); configured != "" {
		return configured
	}
	root, err := os.UserConfigDir()
	if err != nil {
		return "workbraid-data"
	}
	return filepath.Join(root, "workbraid")
}

func originForLoopbackAddress(address string) (string, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return "", fmt.Errorf("invalid listen address %q: %w", address, err)
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return "", fmt.Errorf("listen address must use a literal loopback IP")
	}
	if port == "" || port == "0" {
		return "", fmt.Errorf("listen address must specify a non-zero port")
	}
	return "http://" + net.JoinHostPort(host, port), nil
}
