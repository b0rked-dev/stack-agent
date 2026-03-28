package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Server is a lightweight HTTP server exposing /healthz and /metrics.
type Server struct {
	addr      string
	version   string
	startTime time.Time
	reg       *prometheus.Registry
	mux       *http.ServeMux
}

// New creates a Server. addr is the listen address (e.g. ":2112").
func New(addr, version string, startTime time.Time, reg *prometheus.Registry) *Server {
	s := &Server{
		addr:      addr,
		version:   version,
		startTime: startTime,
		reg:       reg,
		mux:       http.NewServeMux(),
	}
	s.mux.HandleFunc("/healthz", s.healthz)
	s.mux.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
	return s
}

// ServeHTTP implements http.Handler so the server can be used directly in tests
// without binding a real port.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

// Run starts the HTTP server and blocks until ctx is cancelled, then shuts down
// gracefully.
func (s *Server) Run(ctx context.Context) error {
	srv := &http.Server{
		Addr:    s.addr,
		Handler: s.mux,
	}

	ln, err := net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("server: listen %s: %w", s.addr, err)
	}

	errCh := make(chan error, 1)
	go func() {
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
		close(errCh)
	}()

	slog.Info("server: listening", "addr", ln.Addr().String())

	select {
	case <-ctx.Done():
	case err := <-errCh:
		return err
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("server: shutdown: %w", err)
	}
	return <-errCh
}

func (s *Server) healthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	resp := map[string]string{
		"status":  "ok",
		"version": s.version,
		"uptime":  time.Since(s.startTime).String(),
	}
	_ = json.NewEncoder(w).Encode(resp)
}
