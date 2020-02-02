package mocks

import (
	"time"

	"github.com/mainflux/mainflux/internal/pkg/server"
	"github.com/mainflux/mainflux/logger"
)

var _ server.GracefulServer = (*Server)(nil)

// Server mocked server
type Server struct {
	StartErr error
	StopErr  error
}

func (s *Server) Start(logger logger.Logger, errChan chan<- error) {
	logger.Info("mocked server started")

	if s.StartErr != nil {
		errChan <- s.StartErr
		return
	}

}

func (s *Server) Stop(logger logger.Logger, timeout time.Duration) error {
	if s.StopErr == nil {
		logger.Info("mocked server stopped")
		return nil
	}

	return s.StopErr
}
