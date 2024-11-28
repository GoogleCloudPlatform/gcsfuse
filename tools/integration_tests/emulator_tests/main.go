// Copyright 2024 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var (
	// Flags to accept config-file path.
	fConfigPath = flag.String("config-path", "./configs/sample_config.yaml", "Path to the file")

	// Initialized before the server gets started.
	gConfig *Config

	// Initialized before the server gets started.
	gOpManager *OperationManager
)

const PORT = "8020"

type ProxyHandler struct {
	http.Handler
}

func (ph ProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	requestType := deduceRequestType(r)

	targetURL := fmt.Sprintf("%s%s", gConfig.TargetHost, r.RequestURI)
	req, err := http.NewRequest(r.Method, targetURL, r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	for name, values := range r.Header {
		for _, value := range values {
			req.Header.Add(name, value)
		}
	}

	err = handleRequest(req, requestType)
	if err != nil {
		log.Printf("Error in handling the request: %v", err)
	}

	// Send the request to the target server
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respURL, err := resp.Location()
	// Change the resp URL host to proxy server host.
	if err == nil {
		// Parse the original URL.
		u, err := url.Parse(respURL.String())
		if err != nil {
			log.Println("Error parsing URL:", err)
			return
		}

		u.Host = "localhost:" + PORT
		resp.Header.Set("Location", u.String())
	}

	defer resp.Body.Close()

	// Copy headers from the target server's response
	for name, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(name, value)
		}
	}

	// Copy the response body
	w.WriteHeader(resp.StatusCode)
	_, err = io.Copy(w, resp.Body)
	if err != nil {
		log.Printf("Error in coping response body: %v", err)
	}
}

// ProxyServer represents a simple proxy server
type ProxyServer struct {
	port     string
	server   *http.Server
	shutdown chan os.Signal
}

// NewProxyServer creates a new ProxyServer instance
func NewProxyServer(port string) *ProxyServer {
	return &ProxyServer{
		port:     port,
		shutdown: make(chan os.Signal, 1),
	}
}

// Start starts the proxy server.
func (ps *ProxyServer) Start() {
	ps.server = &http.Server{
		Addr:    ":" + ps.port,
		Handler: ProxyHandler{},
	}

	// Start the server in a new goroutine
	go func() {
		log.Printf("Proxy server started on port %s\n", ps.port)
		if err := ps.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Handle graceful shutdown
	signal.Notify(ps.shutdown, syscall.SIGINT, syscall.SIGTERM)
	<-ps.shutdown
	log.Println("Shutting down proxy server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := ps.server.Shutdown(ctx); err != nil {
		log.Fatalf("Proxy server forced to shutdown: %v", err)
	} else {
		log.Println("Proxy server exiting")
	}
}

func main() {
	// Parse the command-line flags
	flag.Parse()

	var err error
	gConfig, err = parseConfigFile(*fConfigPath)
	log.Printf("%+v\n", gConfig)
	if err != nil {
		log.Printf("Parsing error: %v\n", err)
		os.Exit(1)
	}

	gOpManager = NewOperationManager(*gConfig)
	ps := NewProxyServer(PORT)
	ps.Start()
}
