package controllers

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"gorm.io/gorm"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"CodeRewievService/internal/repository"
	"CodeRewievService/internal/services"
)

const (
	defaultReadTimeout  = 15 * time.Second
	defaultWriteTimeout = 15 * time.Second
	defaultIdleTimeout  = 60 * time.Second
	defaultPort         = 8080
	defaultAddress      = "0.0.0.0"
)

type HTTPServer struct {
	httpServer  *http.Server
	mu          *sync.RWMutex
	isRunning   bool
	config      serverConfig
	logger      *slog.Logger
	controllers *controllersRegistry
}

type serverConfig struct {
	address string
	port    int
}

type controllersRegistry struct {
	user        *UserController
	team        *TeamController
	pullRequest *PullRequestController
	statistics  *StatisticsController
}

func NewHTTPServer(logger *slog.Logger, db *gorm.DB, address string, port int) *HTTPServer {
	config := serverConfig{
		address: normalizeAddress(address),
		port:    normalizePort(port),
	}

	repos := initializeRepositories(db)
	svcs := initializeServices(repos, logger)
	ctrls := initializeControllers(svcs, logger)

	return &HTTPServer{
		config:      config,
		logger:      logger,
		mu:          &sync.RWMutex{},
		controllers: ctrls,
	}
}

func (s *HTTPServer) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isRunning {
		return fmt.Errorf("server is already running")
	}

	router := s.createRouter()
	s.httpServer = s.createHTTPServer(router)

	s.logger.Info("Starting HTTP server", "address", s.config.address, "port", s.config.port)

	errCh := make(chan error, 1)
	go s.runServer(errCh)

	s.isRunning = true

	select {
	case err := <-errCh:
		s.isRunning = false
		return fmt.Errorf("server failed to start: %w", err)
	case <-ctx.Done():
		return s.Stop(ctx, 10*time.Second)
	}
}

func (s *HTTPServer) Stop(ctx context.Context, timeout time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.httpServer == nil || !s.isRunning {
		return nil
	}

	s.logger.Info("Initiating server shutdown...")

	shutdownCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
		s.logger.Error("Failed to shutdown server gracefully", "error", err)
		return err
	}

	s.isRunning = false
	s.logger.Info("Server stopped successfully")
	return nil
}

func (s *HTTPServer) createRouter() *chi.Mux {
	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(s.requestLoggingMiddleware)
	s.registerAllRoutes(router)
	return router
}

func (s *HTTPServer) createHTTPServer(handler http.Handler) *http.Server {
	return &http.Server{
		Addr:         fmt.Sprintf("%s:%d", s.config.address, s.config.port),
		Handler:      handler,
		ReadTimeout:  defaultReadTimeout,
		WriteTimeout: defaultWriteTimeout,
		IdleTimeout:  defaultIdleTimeout,
	}
}

func (s *HTTPServer) runServer(errCh chan<- error) {
	if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		errCh <- err
	}
}

func (s *HTTPServer) registerAllRoutes(router *chi.Mux) {
	s.registerUserRoutes(router)
	s.registerTeamRoutes(router)
	s.registerPullRequestRoutes(router)
	s.registerStatisticsRoutes(router)
	s.logger.Info("All HTTP routes registered successfully")
}

func (s *HTTPServer) registerUserRoutes(router *chi.Mux) {
	router.Route("/users", func(r chi.Router) {
		r.Post("/setIsActive", s.controllers.user.SetUserIsActive)
		r.Get("/getReview", s.controllers.user.GetUserReview)
	})
}

func (s *HTTPServer) registerTeamRoutes(router *chi.Mux) {
	router.Route("/team", func(r chi.Router) {
		r.Post("/add", s.controllers.team.CreateTeam)
		r.Get("/get", s.controllers.team.GetTeam)
		r.Post("/deactivate", s.controllers.team.MassDeactivateTeamUsers)
	})
}

func (s *HTTPServer) registerPullRequestRoutes(router *chi.Mux) {
	router.Route("/pullRequest", func(r chi.Router) {
		r.Post("/create", s.controllers.pullRequest.CreatePR)
		r.Post("/merge", s.controllers.pullRequest.MergePR)
		r.Post("/reassign", s.controllers.pullRequest.ReassignPR)
	})
}

func (s *HTTPServer) registerStatisticsRoutes(router *chi.Mux) {
	router.Get("/statistics", s.controllers.statistics.GetAssignmentsStats)
}

func (s *HTTPServer) requestLoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)

		duration := time.Since(start)
		s.logger.Info("HTTP request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", ww.Status(),
			"duration", duration,
			"remoteAddr", r.RemoteAddr,
		)
	})
}

func initializeRepositories(db *gorm.DB) *repositoriesRegistry {
	return &repositoriesRegistry{
		user:        repository.NewUserRepository(db),
		team:        repository.NewTeamRepository(db),
		pullRequest: repository.NewPullRequestRepository(db),
		statistics:  repository.NewStatisticsRepository(db),
	}
}

type repositoriesRegistry struct {
	user        *repository.UserRepository
	team        *repository.TeamRepository
	pullRequest *repository.PullRequestRepository
	statistics  *repository.StatisticsRepository
}

func initializeServices(repos *repositoriesRegistry, logger *slog.Logger) *servicesRegistry {
	return &servicesRegistry{
		user:        services.NewUserService(repos.user),
		team:        services.NewTeamService(repos.team, repos.pullRequest, logger),
		pullRequest: services.NewPullRequestService(repos.pullRequest, repos.user),
		statistics:  services.NewStatisticsService(repos.statistics),
	}
}

type servicesRegistry struct {
	user        *services.UserService
	team        *services.TeamService
	pullRequest *services.PullRequestService
	statistics  *services.StatisticsService
}

func initializeControllers(svcs *servicesRegistry, logger *slog.Logger) *controllersRegistry {
	return &controllersRegistry{
		user:        NewUserController(svcs.user, logger),
		team:        NewTeamController(svcs.team, logger),
		pullRequest: NewPullRequestController(svcs.pullRequest, logger),
		statistics:  NewStatisticsController(svcs.statistics, logger),
	}
}

func normalizeAddress(address string) string {
	if address == "" {
		return defaultAddress
	}
	return address
}

func normalizePort(port int) int {
	if port == 0 {
		return defaultPort
	}
	return port
}
