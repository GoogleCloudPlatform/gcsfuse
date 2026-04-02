// Copyright 2026 Google LLC
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
	"fmt"
	"io"
	"log"
	"net"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/encoding"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// rawCodec is a Codec that passes through raw bytes without any encoding/decoding.
type rawCodec struct{}

func (rawCodec) Marshal(v interface{}) ([]byte, error) {
	out, ok := v.(*[]byte)
	if !ok {
		return nil, fmt.Errorf("failed to marshal, message is %T, want *[]byte", v)
	}
	return *out, nil
}

func (rawCodec) Unmarshal(data []byte, v interface{}) error {
	dst, ok := v.(*[]byte)
	if !ok {
		return fmt.Errorf("failed to unmarshal, message is %T, want *[]byte", v)
	}
	*dst = data
	return nil
}

func (rawCodec) Name() string {
	return "raw-proxy-codec"
}

func init() {
	encoding.RegisterCodec(rawCodec{})
}

// validateGRPCMetadata validates gRPC metadata based on configured header validations
func validateGRPCMetadata(md metadata.MD, validations []HeaderValidation) error {
	for _, validation := range validations {
		headerValues := md.Get(validation.HeaderName)

		if len(headerValues) == 0 {
			if validation.FailOnMismatch {
				return status.Errorf(codes.InvalidArgument, "required metadata %s not found", validation.HeaderName)
			}
			log.Printf("Warning: metadata %s not found", validation.HeaderName)
			continue
		}

		// Check if any of the header values contain the expected pattern
		found := false
		for _, headerValue := range headerValues {
			if validation.ExpectedPattern != "" && strings.Contains(headerValue, validation.ExpectedPattern) {
				found = true
				log.Printf("Metadata validation passed: %s = %s (contains '%s')", validation.HeaderName, headerValue, validation.ExpectedPattern)
				break
			}
		}

		if validation.ExpectedPattern != "" && !found {
			err := fmt.Errorf("metadata %s values %v do not contain expected pattern '%s'",
				validation.HeaderName, headerValues, validation.ExpectedPattern)
			if validation.FailOnMismatch {
				return status.Errorf(codes.InvalidArgument, "%v", err)
			}
			log.Printf("Warning: %v", err)
		}
	}
	return nil
}

// startGRPCProxy creates a transparent gRPC proxy that validates metadata and forwards all requests to the target
func startGRPCProxy(listener net.Listener, targetHost string, validations []HeaderValidation) error {
	// Create connection to target for forwarding with raw codec to avoid unmarshaling
	targetConn, err := grpc.NewClient(
		targetHost,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.ForceCodec(rawCodec{})),
	)
	if err != nil {
		return fmt.Errorf("failed to connect to target %s: %w", targetHost, err)
	}

	// Create unknown service handler that validates metadata and forwards all calls
	unknownHandler := func(srv interface{}, stream grpc.ServerStream) error {
		fullMethodName, ok := grpc.MethodFromServerStream(stream)
		if !ok {
			return status.Errorf(codes.Internal, "failed to get method name")
		}

		// Log the gRPC call
		log.Printf("=== Proxying Call: %s ===", fullMethodName)

		// Extract and validate metadata
		md, ok := metadata.FromIncomingContext(stream.Context())
		if ok {
			if err := validateGRPCMetadata(md, validations); err != nil {
				log.Printf("Metadata validation failed: %v", err)
				return err
			}
		} else {
			md = metadata.New(nil)
			if len(validations) > 0 {
				log.Println("Warning: No metadata found in request")
			}
		}

		// Inject testbench instructions via gOpManager
		parts := strings.Split(fullMethodName, "/")
		methodName := parts[len(parts)-1]
		plantOp := gOpManager.retrieveOperation(RequestType(methodName))
		if plantOp != "" {
			if *fDebug {
				log.Printf("Planting operation: %s for method: %s", plantOp, fullMethodName)
			}
			// Inject direct instruction for testbench
			md = metadata.Join(md, metadata.Pairs("x-goog-emulator-instructions", plantOp))
		}

		// Forward metadata to target
		forwardCtx := metadata.NewOutgoingContext(stream.Context(), md)

		// Invoke the method on the target
		clientStream, err := targetConn.NewStream(forwardCtx, &grpc.StreamDesc{
			StreamName:    fullMethodName,
			ServerStreams: true,
			ClientStreams: true,
		}, fullMethodName, grpc.ForceCodec(rawCodec{}))
		if err != nil {
			return status.Errorf(codes.Internal, "failed to create target stream: %v", err)
		}

		// Proxy messages bidirectionally using raw bytes
		// Create channels for error handling
		clientToServer := make(chan error, 1)
		serverToClient := make(chan error, 1)

		// Receive from client, send to target
		go func() {
			for {
				req := new([]byte)
				if err := stream.RecvMsg(req); err != nil {
					if err == io.EOF {
						err = clientStream.CloseSend()
						clientToServer <- err
						return
					}
					clientToServer <- fmt.Errorf("receiving from client: %w", err)
					return
				}
				if err := clientStream.SendMsg(req); err != nil {
					clientToServer <- fmt.Errorf("sending to server: %w", err)
					return
				}
			}
		}()

		// Receive from target, send to client
		go func() {
			for {
				resp := new([]byte)
				if err := clientStream.RecvMsg(resp); err != nil {
					if err == io.EOF {
						serverToClient <- nil
						return
					}
					serverToClient <- fmt.Errorf("receiving from server: %w", err)
					return
				}
				if err := stream.SendMsg(resp); err != nil {
					serverToClient <- fmt.Errorf("sending to client: %w", err)
					return
				}
			}
		}()

		// Wait for BOTH directions to complete
		// This is important for both unary and streaming RPCs
		var err1, err2 error
		for i := 0; i < 2; i++ {
			select {
			case err := <-clientToServer:
				if err != nil {
					log.Printf("Client to server error: %v", err)
					err1 = err
				}
			case err := <-serverToClient:
				if err != nil {
					log.Printf("Server to client error: %v", err)
					err2 = err
				}
			}
		}

		// Return first error encountered, or nil if both succeeded
		if err1 != nil {
			return err1
		}
		return err2
	}

	// Create gRPC server with unknown service handler and raw codec
	opts := []grpc.ServerOption{
		grpc.UnknownServiceHandler(unknownHandler),
		grpc.ForceServerCodec(rawCodec{}),
	}
	grpcServer := grpc.NewServer(opts...)

	log.Printf("gRPC proxy server ready to accept connections, forwarding to %s", targetHost)

	return grpcServer.Serve(listener)
}
