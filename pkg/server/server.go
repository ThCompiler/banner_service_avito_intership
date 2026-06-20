package server

import (
	"context"
	"net/http"
	"time"

	"github.com/ThCompiler/sdi"
)

const (
	defaultReadTimeout     = 5 * time.Second
	defaultWriteTimeout    = 5 * time.Second
	defaultAddr            = ":80"
	defaultShutdownTimeout = 3 * time.Second
)

type Server struct {
	server          *http.Server
	notify          chan error
	shutdownTimeout time.Duration
}

type Config struct {
	Port string
}

type ProviderDeps struct {
	Config  Config
	Handler http.Handler
}

func New(server http.Handler, opts ...Option) *Server {
	httpServer := &http.Server{
		Handler:      server,
		ReadTimeout:  defaultReadTimeout,
		WriteTimeout: defaultWriteTimeout,
		Addr:         defaultAddr,
	}

	s := &Server{
		server:          httpServer,
		notify:          make(chan error, 1),
		shutdownTimeout: defaultShutdownTimeout,
	}

	// Custom options
	for _, opt := range opts {
		opt(s)
	}

	s.start()

	return s
}

func (s *Server) start() {
	go func() {
		s.notify <- s.server.ListenAndServe()
		close(s.notify)
	}()
}

func (s *Server) Notify() <-chan error {
	return s.notify
}

func (s *Server) Shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
	defer cancel()

	return s.server.Shutdown(ctx)
}

func NewProvider() sdi.Provider[*Server, ProviderDeps] {
	return sdi.ProviderFuncNoClean(func(_ context.Context, deps ProviderDeps) (*Server, error) {
		return New(deps.Handler, Port(deps.Config.Port)), nil
	})
}
