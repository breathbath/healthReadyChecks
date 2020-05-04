package rest

import (
	"context"
	"errors"
	"fmt"
	"github.com/breathbath/healthz/health"
	"github.com/breathbath/healthz/logging"
	"github.com/breathbath/healthz/ready"
	"github.com/gorilla/mux"
	"net/http"
	"time"
)

const portToUse = 9244

//Server wraps health/ready http server implementation
type Server struct {
	readyChecker  ready.Checker
	readyTimeout  time.Duration
	healthChecker health.Checker
	isWithReady   bool
	isWithHealth  bool
}

//WithHealth returns Server with initialised health functionality
func WithHealth(s Server, healthChecker health.Checker) Server {
	s.healthChecker = healthChecker
	s.isWithHealth = true

	return s
}

//WithReady returns Server with initialised ready functionality
func WithReady(s Server, readyChecker ready.Checker, readyTimeout time.Duration) Server {
	s.readyChecker = readyChecker
	s.readyTimeout = readyTimeout
	s.isWithReady = true

	return s
}

//Start starts health or/and ready server if they were initialized, if not returns an error
func (s Server) Start(ctx context.Context, targetPort int) error {
	if !s.isWithReady && !s.isWithHealth {
		return errors.New("neither ready nor health logic was initialised")
	}
	router := mux.NewRouter().StrictSlash(false)

	if s.isWithHealth {
		logging.L.InfoF("Will start health listener with healthz api")
		router.Handle("/healthz", NewHealthHandler(s.healthChecker))
	}

	if s.isWithReady {
		logging.L.InfoF("Will start ready listener with readyz api")
		router.Handle("/readyz", NewReadyHandler(s.readyTimeout, s.readyChecker))
	}

	addr := fmt.Sprintf(":%d", targetPort)
	httpServer := http.Server{
		Addr:    addr,
		Handler: router,
	}

	logging.L.InfoF("Starting health/ready REST server at %s", addr)

	go func(s http.Server, c context.Context) {
		<-c.Done()
		logging.L.InfoF("Exiting health REST server at %s", addr)

		err := s.Close()
		if err != nil {
			logging.L.ErrorF(err.Error())
		} else {
			logging.L.InfoF("Exit success for health REST %s", addr)
		}
	}(httpServer, ctx)

	return httpServer.ListenAndServe()
}

//NewReadyHandler gives http.Handler implementation for readiness checks
func NewReadyHandler(readyTimeout time.Duration, readyChecker ready.Checker) http.Handler{
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		readyCtx, cancelReady := context.WithTimeout(context.Background(), readyTimeout)
		defer cancelReady()

		isReady, err := readyChecker.IsReady(readyCtx)
		if isReady {
			w.WriteHeader(http.StatusOK)
			return
		}

		w.WriteHeader(http.StatusInternalServerError)
		if err == nil {
			return
		}
		_, err = w.Write([]byte(err.Error()))
		if err != nil {
			logging.L.ErrorF("Failed to write body: %v", err)
		}
	})
}

//NewHealthHandler gives http.Handler implementation for health checks
func NewHealthHandler(healthChecker health.Checker) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		isHealthy, unhealthyReason := healthChecker.IsHealthy()
		if isHealthy {
			w.WriteHeader(http.StatusOK)
			return
		}

		w.WriteHeader(http.StatusInternalServerError)
		_, err := w.Write([]byte(unhealthyReason))
		if err != nil {
			logging.L.ErrorF("Failed to write body: %v", err)
		}
	})
}
