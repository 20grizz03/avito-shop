package main

import (
	"context"
	"github.com/linemk/avito-shop/internal/service"
	"github.com/linemk/avito-shop/internal/storage"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/linemk/avito-shop/internal/app"
	"github.com/linemk/avito-shop/internal/app/handlers"
	"github.com/linemk/avito-shop/internal/config"
	"github.com/linemk/avito-shop/internal/lib/logger"
	"github.com/linemk/avito-shop/internal/lib/logger/handlers/urllog"
)

func main() {
	// загрузка конфигурации
	cfg := config.MustLoad()

	// инициализация логгера, зависит от настройки окружения
	log := logger.SetupLogger(cfg.Env)

	log.Info("Starting server", slog.String("env", cfg.Env))
	log.Debug("Debug messages are enabled")

	// загружаем объект приложения с логгером, конфигом и подключением к БД
	application, err := app.NewApp(log, cfg)
	if err != nil {
		slog.Error("failed to initialize app", err)
		os.Exit(1)
	}
	defer application.DB.Close()

	router := chi.NewRouter()
	// настройка middleware
	router.Use(middleware.RequestID)
	router.Use(urllog.CustomLoggerMiddleware(log))
	router.Use(middleware.Recoverer)
	router.Use(middleware.URLFormat)

	userRepo := storage.NewUserRepository(application.DB)
	authService := service.NewAuthService(application.Logger, userRepo, time.Duration(application.Config.JWT.TokenTTL)*time.Minute)

	router.Post("/api/auth", handlers.AuthHandler(application.Logger, authService))

	router.Group(func(r chi.Router) {
		// Здесь подключаем middleware, который проверяет JWT-токен
		//jwtMW := jwtmiddleware.NewJWTMiddleware(cfg.JWT.Secret)
		//r.Use(jwtMW.VerifyToken)

		//// Эндпоинт для получения информации о монетах, инвентаре и истории транзакций
		//r.Get("/api/info", handlers.InfoHandler(log))
		//
		//// Эндпоинт для отправки монет другому пользователю
		//r.Post("/api/sendCoin", handlers.SendCoinHandler(log))
		//
		//// Эндпоинт для покупки мерча (параметр в path — название товара)
		//r.Get("/api/buy/{item}", handlers.BuyHandler(log))
	})

	log.Info("starting server", slog.String("address", cfg.HTTPServer.Address))

	srv := &http.Server{
		Addr:         cfg.HTTPServer.Address,
		Handler:      router,
		ReadTimeout:  cfg.HTTPServer.Timeout,
		WriteTimeout: cfg.HTTPServer.Timeout,
		IdleTimeout:  cfg.HTTPServer.IdleTimeout,
	}

	go func() {
		log.Info("starting server", slog.String("address", cfg.HTTPServer.Address))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("server error", slog.Any("error", err))
		}
	}()

	// graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	stopSign := <-stop
	log.Info("received shutdown signal", slog.String("signal", stopSign.String()))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error("server shutdown failed", slog.Any("error", err))
	}
	log.Info("server gracefully stopped")
}
