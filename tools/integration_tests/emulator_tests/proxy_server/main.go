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
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

const PortAndProxyProcessIdInfoLogFormat = "Listening Proxy Server On Port [%s] with Process ID [%d]"

var (
	// Flag to accept config-file path.
	fConfigPath = flag.String("config-path", "configs/config.yaml", "Path to the file")
	// Flag to turn on/off fDebug logs.
	fDebug = flag.Bool("debug", true, "Enable proxy server fDebug logs.")
	// Log file to write proxy server logs.
	fLogFilePath = flag.String("log-file", "", "Path to the log file")
	// Initialized before the server gets started.
	gConfig    *Config
	gOpManager *OperationManager
	// Port number assigned to listener.
	gPort string
)

type ProxyHandler struct {
	http.Handler
}

// logRequestAndType is used for logging the request on proxy server.
// More fields can be added or removed as per requirement for debugging purpose.
func logRequestAndType(req *http.Request, r RequestType) {
	// Print empty lines to separate each request in log.
	log.Println("")
	log.Println("")
	log.Printf("RequestType: %s\n", r)
	log.Printf("URL: %s\n", req.URL.String())
	log.Printf("Content-Length: %s\n", req.Header.Get("Content-Length"))
	log.Printf("Content-Range: %s\n", req.Header.Get("Content-Range"))
}

// AddRetryID creates mock error behavior on the target host for specific request types.
// It retrieves the corresponding operation from the operation manager based on the provided RequestTypeAndInstruction.
// If a matching operation is found, it creates a retry test with the target host and instruction,
// and attaches the generated test ID to the HTTP request header "x-retry-test-id".
//
// This function is used to simulate error scenarios for testing retry mechanisms.
func AddRetryID(req *http.Request, r RequestTypeAndInstruction) error {
	plantOp := gOpManager.retrieveOperation(r.RequestType)
	if *fDebug {
		logRequestAndType(req, r.RequestType)
		if plantOp != "" {
			log.Println("Planting operation: ", plantOp)
		}
	}
	if plantOp != "" {
		testID, err := CreateRetryTest(gConfig.TargetHost, map[string][]string{r.Instruction: {plantOp}})
		if err != nil {
			return fmt.Errorf("CreateRetryTest: %v", err)
		}
		req.Header.Set("x-retry-test-id", testID)
	}
	return nil
}

// ServeHTTP handles incoming HTTP requests. It acts as a proxy, forwarding requests
// to a target server specified in the configuration and then relaying the
// response back to the original client.
func (ph ProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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
	// Determine the request type and instruction (e.g., read, write, metadata) based on the incoming request.
	reqTypeAndInstruction := deduceRequestTypeAndInstruction(r)

	// Add a unique retry ID to the request headers, associating it with the
	// deduced request type and instruction. This is used for adding custom failures on requests.
	err = AddRetryID(req, reqTypeAndInstruction)
	if err != nil {
		log.Printf("AddRetryID: %v", err)
	}

	// Send the request to the target server
	client := &http.Client{}
	start := time.Now()
	resp, err := client.Do(req)
	elapsed := time.Since(start)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respURL, err := resp.Location()
	// Change the response URL host to the proxy server host.
	// This is necessary because, from the client's perspective, the proxy server is the endpoint.
	// Therefore, the response must appear to originate from the proxy host.
	if err == nil {
		// Parse the original URL.
		u, err := url.Parse(respURL.String())
		if err != nil {
			log.Println("Error parsing URL:", err)
			return
		}

		u.Host = "localhost:" + gPort
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
	if *fDebug {
		log.Printf("Respnse Status: %d\n", resp.StatusCode)
		log.Printf("Elapsed Time: %.3fs\n", elapsed.Seconds())
	}
}

// ProxyServer represents a simple proxy server over GCS storage based API endpoint.
type ProxyServer struct {
	server   *http.Server
	shutdown chan os.Signal
}

// NewProxyServer creates a new ProxyServer instance
func NewProxyServer() *ProxyServer {
	return &ProxyServer{
		shutdown: make(chan os.Signal, 1),
	}
}

// Start starts the proxy server.
func (ps *ProxyServer) Start() {
	//  Create a listener on random available port.
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		log.Fatalf("Error on listening: %v", err)
	}
	gPort = strconv.Itoa(listener.Addr().(*net.TCPAddr).Port)
	// Log port number and proxy process Id for the proxy server.
	log.Printf(PortAndProxyProcessIdInfoLogFormat, gPort, os.Getpid())
	ps.server = &http.Server{
		Addr:    ":" + gPort,
		Handler: ProxyHandler{},
	}

	// Start the server in a new goroutine
	go func() {
		if err := ps.server.Serve(listener); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Handle graceful shutdown
	signal.Notify(ps.shutdown, syscall.SIGINT, syscall.SIGTERM)
	// Blocks until one of the Signal is recieved.
	<-ps.shutdown
	log.Println("Shutting down proxy server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
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
	if err != nil {
		log.Printf("Parsing error: %v\n", err)
		os.Exit(1)
	}

	if *fLogFilePath == "" {
		log.Println("No log file path for proxy server provided.")
		os.Exit(1)
	}
	logFile, err := os.OpenFile(*fLogFilePath, os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Printf("Error opening log file: %v\n", err)
		os.Exit(1)
	}
	defer logFile.Close()
	log.SetOutput(logFile)

	if *fDebug {
		printConfig(*gConfig)
	}

	gOpManager = NewOperationManager(*gConfig)

	ps := NewProxyServer()
	ps.Start()
}
