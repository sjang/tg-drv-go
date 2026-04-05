package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"

	tgclient "tg-drv-go/internal/telegram"
)

type Server struct {
	router *chi.Mux
	tg     *tgclient.Client
	logger *zap.Logger
	port   int
	server *http.Server
}

func NewServer(tg *tgclient.Client, port int, logger *zap.Logger) *Server {
	s := &Server{
		tg:     tg,
		logger: logger,
		port:   port,
	}
	s.setupRoutes()
	return s
}

func (s *Server) SetClient(tg *tgclient.Client) {
	s.tg = tg
}

func (s *Server) setupRoutes() {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(corsMiddleware)

	r.Route("/api", func(r chi.Router) {
		// Auth
		r.Post("/auth/send-code", s.handleSendCode)
		r.Post("/auth/verify", s.handleVerifyCode)
		r.Post("/auth/verify-password", s.handleVerifyPassword)
		r.Get("/auth/status", s.handleAuthStatus)

		// Folders
		r.Get("/folders", s.handleListFolders)
		r.Post("/folders", s.handleCreateFolder)
		r.Put("/folders/{id}", s.handleRenameFolder)
		r.Delete("/folders/{id}", s.handleDeleteFolder)
		r.Post("/folders/sync", s.handleSyncFolders)

		// Files
		r.Get("/folders/{id}/files", s.handleListFiles)
		r.Post("/folders/{id}/files/upload", s.handleUploadFile)

		r.Get("/files/{id}/download", s.handleDownloadFile)
		r.Get("/files/{id}/stream", s.handleStreamFile)
		r.Get("/files/{id}/thumbnail", s.handleThumbnail)
		r.Get("/files/{id}/player", s.handlePlayer)
		r.Put("/files/{id}", s.handleRenameFile)
		r.Delete("/files/{id}", s.handleDeleteFile)

		// Index rebuild
		r.Post("/folders/{id}/rebuild", s.handleRebuildIndex)
	})

	s.router = r
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) Start() error {
	addr := fmt.Sprintf("127.0.0.1:%d", s.port)
	s.server = &http.Server{
		Addr:    addr,
		Handler: s.router,
	}
	s.logger.Info("starting API server", zap.String("addr", addr))
	return s.server.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	if s.server != nil {
		return s.server.Shutdown(ctx)
	}
	return nil
}

func (s *Server) Port() int {
	return s.port
}

func (s *Server) Handler() http.Handler {
	return s.router
}
