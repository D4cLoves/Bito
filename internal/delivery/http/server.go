package http

import (
	"context"
	"log"
	"net/http"
	"time"

	"bito/internal/delivery/ws"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

type Server struct {
	httpServer    *http.Server
	hub           *ws.Hub
	router        *chi.Mux
	authHandler   *AuthHandler
	tableHandler  *TableHandler
	userHandler   *UserHandler
	walletHandler *WalletHandler
	gameHandler   *GameHandler
	tokens        TokenValidator
}

type Handlers struct {
	Auth   *AuthHandler
	Table  *TableHandler
	User   *UserHandler
	Wallet *WalletHandler
	Game   *GameHandler
}

func NewServer(addr string, hub *ws.Hub, h Handlers, tokens TokenValidator) *Server {
	s := &Server{
		hub:           hub,
		router:        chi.NewRouter(),
		authHandler:   h.Auth,
		tableHandler:  h.Table,
		userHandler:   h.User,
		walletHandler: h.Wallet,
		gameHandler:   h.Game,
		tokens:        tokens,
	}

	s.setupMiddlewares()
	s.setupRoutes()

	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      s.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return s
}

func (s *Server) setupMiddlewares() {
	s.router.Use(middleware.RequestID)
	s.router.Use(middleware.RealIP)
	s.router.Use(middleware.Logger)
	s.router.Use(middleware.Recoverer)

	s.router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300,
	}))
}

func (s *Server) setupRoutes() {
	s.router.Get("/ws", s.hub.ServeWS)

	s.router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	if s.authHandler != nil {
		s.router.Route("/api/v1/auth", func(r chi.Router) {
			r.Post("/send-code", s.authHandler.SendCode)
			r.Post("/verify-code", s.authHandler.VerifyCode)
			r.Post("/register", s.authHandler.VerifyCode)
			r.Post("/login", s.authHandler.Login)
			r.Post("/refresh", s.authHandler.RefreshToken)
			r.Post("/logout", s.authHandler.Logout)
		})
	}

	if s.tableHandler != nil {
		s.router.Route("/api/v1/tables", func(r chi.Router) {
			r.Use(AuthMiddleware(s.tokens))
			r.Post("/", s.tableHandler.Create)
			r.Post("/quick", s.tableHandler.Quick)
			r.Get("/", s.tableHandler.List)
			r.Get("/{id}", s.tableHandler.Get)
		})
	}

	if s.userHandler != nil || s.tableHandler != nil || s.gameHandler != nil {
		s.router.Route("/api/v1/users/me", func(r chi.Router) {
			r.Use(AuthMiddleware(s.tokens))
			if s.userHandler != nil {
				r.Get("/", s.userHandler.Profile)
				r.Patch("/", s.userHandler.ChangeUsername)
				r.Delete("/", s.userHandler.DeleteAccount)
				r.Post("/password", s.userHandler.ChangePassword)
			}
			if s.tableHandler != nil {
				r.Get("/active-table", s.tableHandler.Active)
			}
			if s.gameHandler != nil {
				r.Get("/games", s.gameHandler.History)
			}
		})
	}

	if s.walletHandler != nil {
		s.router.Route("/api/v1/wallet", func(r chi.Router) {
			r.Use(AuthMiddleware(s.tokens))
			r.Post("/top-up", s.walletHandler.TopUp)
			r.Get("/transactions", s.walletHandler.Transactions)
		})
	}
}

func (s *Server) Router() *chi.Mux {
	return s.router
}

func (s *Server) Start() error {
	log.Printf("HTTP server listening on %s", s.httpServer.Addr)
	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}
