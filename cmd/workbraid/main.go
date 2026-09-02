package main

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr, os.Stdin))
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
