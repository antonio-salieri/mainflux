// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/mainflux/mainflux/logger"
)

// HTTPServer wraper around http.Server that fulfill GracefulServer interface
type HTTPServer struct {
	*http.Server
	certPath string
	keyPath  string
}

var _ GracefulServer = (*HTTPServer)(nil)

// NewHTTPServer creates new http server
func NewHTTPServer(address string, handler http.Handler, certPath string, keyPath string) *HTTPServer {
	return &HTTPServer{
		Server: &http.Server{
			Addr:    address,
			Handler: handler,
		},
		certPath: certPath,
		keyPath:  keyPath,
	}
}

// Start starts http or https server, depending on cert/key value
func (s *HTTPServer) Start(log logger.Logger, errs chan<- error) {
	if s.certPath != "" || s.keyPath != "" {
		log.Info(fmt.Sprintf("HTTPS service started on address %s, cert %s key %s", s.Addr, s.certPath, s.keyPath))
		errs <- s.ListenAndServeTLS(s.certPath, s.keyPath)
		return
	}
	log.Info(fmt.Sprintf("HTTP service started on address %s", s.Addr))
	errs <- s.ListenAndServe()
}

// Stop stops http listenning and waits for existing handler calls to complete or timeout to occur
func (s *HTTPServer) Stop(log logger.Logger, timeout time.Duration) error {
	log.Info(fmt.Sprintf("Stopping http server on %q", s.Addr))
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := s.Shutdown(ctx); err == context.DeadlineExceeded {
		log.Error("HTTP shutdown timeout - existing connections are not completed yet")
	} else if err != nil {
		return fmt.Errorf("Error stopping http server: %s", err)
	}

	log.Info(fmt.Sprintf("HTTP server on %q stopped", s.Addr))
	return nil
}
