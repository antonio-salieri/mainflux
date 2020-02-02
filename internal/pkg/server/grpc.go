// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/mainflux/mainflux/logger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// GRPCServer wrapper around grpc.Sserver that fullfils GracefulServer interface
type GRPCServer struct {
	*grpc.Server
	address string
}

var _ GracefulServer = (*GRPCServer)(nil)

// NewGRPCServer creates new GRPCServer
func NewGRPCServer(address string, certPath string, keyPath string, log logger.Logger) *GRPCServer {
	srv := &GRPCServer{address: address}

	if certPath != "" || keyPath != "" {
		creds, err := credentials.NewServerTLSFromFile(certPath, keyPath)
		if err != nil {
			panic(fmt.Sprintf("Failed to load authn certificates: %s", err))
		}
		log.Info(fmt.Sprintf("gRPC service created using https with cert %s key %s", certPath, keyPath))
		srv.Server = grpc.NewServer(grpc.Creds(creds))
	} else {
		log.Info(fmt.Sprintf("gRPC service created using http"))
		srv.Server = grpc.NewServer()
	}

	return srv
}

// Start starts GRPC listenner
func (s *GRPCServer) Start(log logger.Logger, errs chan<- error, registerServiceFn func()) {
	if registerServiceFn == nil {
		errs <- fmt.Errorf("Register service function is nil")
		return
	}

	// Bind service to grpc server
	registerServiceFn()

	listener, err := net.Listen("tcp", s.address)
	if err != nil {
		errs <- fmt.Errorf("Failed to listen on address %s: %s", s.address, err)
		return
	}

	log.Info(fmt.Sprintf("gRPC service started on %s", s.address))
	errs <- s.Serve(listener)
}

// Stop stops listenning for new GRPC connections and waits for existing service calls to complete or timeout to exceed
func (s *GRPCServer) Stop(log logger.Logger, timeout time.Duration) error {
	log.Info(fmt.Sprintf("Stopping gRPC server on %q", s.address))

	serverStopped := make(chan struct{})
	go func() {
		s.GracefulStop()
		close(serverStopped)
	}()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
loop:
	for {
		select {
		case <-serverStopped:
			break loop
		case <-ctx.Done():
			log.Error(fmt.Sprintf("Stop timeout of %v exceeded - some calls are still active", timeout))
			break loop
		}
	}

	log.Info(fmt.Sprintf("gRPC server on %q stopped", s.address))
	return nil
}
