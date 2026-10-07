package app

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"bito/internal/config"
	deliveryHTTP "bito/internal/delivery/http"
	"bito/internal/delivery/ws"
	"bito/internal/pkg/jwt"
	"bito/internal/pkg/mailer"
	"bito/internal/repository/postgres"
	redisRepo "bito/internal/repository/redis"
	"bito/internal/service"
)

func Run() {
	// 1. Контекст
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 2. Конфигурация (.env)
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// 3. Пул соединений PostgreSQL
	pool, err := postgres.NewDB(ctx, cfg)
	if err != nil {
		log.Fatalf("failed to connect to postgres: %v", err)
	}
	defer pool.Close()

	// 4. Клиент и репозиторий Redis
	redisClient, err := redisRepo.New(ctx, cfg)
	if err != nil {
		log.Fatalf("failed to connect to redis: %v", err)
	}
	defer redisClient.Close()
	authRedisRepo := redisRepo.NewAuthRepository(redisClient)

	// 5. Репозиторий пользователей
	userRepo := postgres.NewUserRepository(pool)

	// 6. JWT Менеджер
	tokenManager := jwt.NewTokenManager(cfg.JWTSecret, cfg.JWTAccessTTL, cfg.JWTRefreshTTL)

	// 7. Почтовый сервис и генератор OTP
	mailerClient := mailer.New(cfg)
	otpService := service.NewOTPService()

	// 8. Бизнес-логика (AuthService)
	authService := service.NewAuthService(userRepo, otpService, mailerClient, authRedisRepo)

	// 9. HTTP Хендлер
	authHandler := deliveryHTTP.NewAuthHandler(authService, tokenManager, cfg.JWTRefreshTTL)

	// 8. WebSocket Hub
	hub := ws.NewHub()

	// 9. HTTP Сервер (твой конструктор с настроенными роутами /api/v1/auth)
	serverAddr := fmt.Sprintf(":%s", cfg.ServerPort)
	server := deliveryHTTP.NewServer(serverAddr, hub, authHandler)

	// Запуск сервера в отдельной горутине
	go func() {
		if err := server.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("HTTP server failed: %v", err)
		}
	}()

	log.Printf("Bito Server started on port %s", cfg.ServerPort)

	// 10. Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	sig := <-quit

	log.Printf("Received signal %v. Shutting down...", sig)

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	log.Println("Bito Server gracefully stopped.")
}
