// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package server_test

import (
	"errors"
	"os"
	"syscall"
	"time"

	"github.com/mainflux/mainflux/internal/pkg/server"
	"github.com/mainflux/mainflux/internal/pkg/server/mocks"
	log "github.com/mainflux/mainflux/logger/mocks"
)

var (
	errChan = make(chan error)
	logger  = log.Logger{}
)

func ExampleMonitor_noServerError() {
	srv := &mocks.Server{}
	go srv.Start(logger, errChan)
	time.Sleep(50 * time.Millisecond) // ensure Start is called
	go func() {
		_ = syscall.Kill(os.Getpid(), syscall.SIGINT)
	}()
	server.Monitor(logger, errChan, srv)

	// Output:
	// Info: mocked server started
	// Info: Graceful server monitor started
	// Info: Received signal: interrupt
	// Info: mocked server stopped
	// Info: Graceful server monitor exit
}

func ExampleMonitor_errorWhileRunning() {
	srv := &mocks.Server{StartErr: errors.New("server error")}
	go srv.Start(logger, errChan)
	server.Monitor(logger, errChan, srv)

	// Output:
	// Info: Graceful server monitor started
	// Info: mocked server started
	// Error: Graceful server monitor exiting due to error: server error
}

func ExampleMonitor_errorStoppingServer() {
	srv := &mocks.Server{StopErr: errors.New("error stopping server")}
	go srv.Start(logger, errChan)
	time.Sleep(50 * time.Millisecond) // ensure Start is called
	go func() {
		_ = syscall.Kill(os.Getpid(), syscall.SIGINT)
	}()
	server.Monitor(logger, errChan, srv)

	// Output:
	// Info: mocked server started
	// Info: Graceful server monitor started
	// Info: Received signal: interrupt
	// Error: Error during server stop: error stopping server
	// Info: Graceful server monitor exit
}
