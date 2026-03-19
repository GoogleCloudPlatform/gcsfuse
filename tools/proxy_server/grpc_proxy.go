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
	"fmt"
	"log"
	"net"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

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

// unaryInterceptor intercepts unary gRPC calls for metadata validation
func unaryInterceptor(validations []HeaderValidation) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Extract and validate metadata
		md, ok := metadata.FromIncomingContext(ctx)
		if ok {
			if err := validateGRPCMetadata(md, validations); err != nil {
				log.Printf("Metadata validation failed: %v", err)
				return nil, err
			}
		} else if len(validations) > 0 {
			log.Println("Warning: No metadata found in request")
		}

		// Forward the call to the handler
		return handler(ctx, req)
	}
}

// streamInterceptor intercepts streaming gRPC calls for metadata validation
func streamInterceptor(validations []HeaderValidation) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		// Extract and validate metadata
		md, ok := metadata.FromIncomingContext(ss.Context())
		if ok {
			if err := validateGRPCMetadata(md, validations); err != nil {
				log.Printf("Metadata validation failed: %v", err)
				return err
			}
		} else if len(validations) > 0 {
			log.Println("Warning: No metadata found in stream")
		}

		// Forward the call to the handler
		return handler(srv, ss)
	}
}

// startGRPCProxy creates a transparent gRPC proxy that validates metadata and forwards all requests to the target
func startGRPCProxy(listener net.Listener, targetHost string, validations []HeaderValidation) error {
	// Create connection to target for forwarding
	targetConn, err := grpc.NewClient(targetHost, grpc.WithTransportCredentials(insecure.NewCredentials()))
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
		} else if len(validations) > 0 {
			log.Println("Warning: No metadata found in request")
		}

		// Forward metadata to target
		forwardCtx := metadata.NewOutgoingContext(stream.Context(), md)

		// Invoke the method on the target
		clientStream, err := targetConn.NewStream(forwardCtx, &grpc.StreamDesc{
			StreamName:    fullMethodName,
			ServerStreams: true,
			ClientStreams: true,
		}, fullMethodName)
		if err != nil {
			return status.Errorf(codes.Internal, "failed to create target stream: %v", err)
		}

		// Proxy messages bidirectionally (simplified - handles unary as special case of stream)
		// Receive from client, send to target
		go func() {
			for {
				var req interface{}
				if err := stream.RecvMsg(&req); err != nil {
					clientStream.CloseSend()
					return
				}
				if err := clientStream.SendMsg(req); err != nil {
					return
				}
			}
		}()

		// Receive from target, send to client
		for {
			var resp interface{}
			if err := clientStream.RecvMsg(&resp); err != nil {
				return err
			}
			if err := stream.SendMsg(resp); err != nil {
				return err
			}
		}
	}

	// Create gRPC server with interceptors and unknown service handler
	opts := []grpc.ServerOption{
		grpc.UnaryInterceptor(unaryInterceptor(validations)),
		grpc.StreamInterceptor(streamInterceptor(validations)),
		grpc.UnknownServiceHandler(unknownHandler),
	}
	grpcServer := grpc.NewServer(opts...)

	log.Printf("gRPC proxy server ready to accept connections, forwarding to %s", targetHost)

	return grpcServer.Serve(listener)
}
