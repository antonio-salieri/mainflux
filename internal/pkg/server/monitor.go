// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mainflux/mainflux/logger"
)

const defaultServerStopTimeout = 30 * time.Second

var monitorSignals = []os.Signal{syscall.SIGINT}

// GracefulServer interface with method to start and gracefully stop server
type GracefulServer interface {
	// Stop terminates server listenning and waits for existing handler calls to complete or timeout to occur
	Stop(logger logger.Logger, timeout time.Duration) error
}

// Monitor is blocking function that listens for SIGINT signal and errors on errChan channel.
// When SIGINT signal is received it tries to gracefully stop all passed servers within passed timeout and then terminates monitoring.
// If error is received on errors channel error is logged and monitoring is terminated.
func Monitor(log logger.Logger, errChan <-chan error, servers ...GracefulServer) {
	log.Info("Graceful server monitor started")

	signalChan := make(chan os.Signal, len(monitorSignals))
	signal.Notify(signalChan, monitorSignals...)

	for {
		select {
		case receivedSignal := <-signalChan:
			log.Info(fmt.Sprintf("Received signal: %s", receivedSignal))
			for _, srv := range servers {
				if err := srv.Stop(log, defaultServerStopTimeout); err != nil {
					log.Error(fmt.Sprintf("Error during server stop: %s", err))
				}
			}
			log.Info("Graceful server monitor exit")
			return
		case err := <-errChan:
			log.Error(fmt.Sprintf("Graceful server monitor exiting due to error: %s", err))
			return
		}
	}
}
