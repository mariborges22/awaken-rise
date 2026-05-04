package main

import (
	"database/sql"
	"log/slog"
	"net/http"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/redis/go-redis/v9"
	
	"github.com/wepink-clone/backend/internal/adapters/mysql"
	"github.com/wepink-clone/backend/internal/adapters/payment"
	"github.com/wepink-clone/backend/internal/adapters/rabbitmq"
	redisAdapter "github.com/wepink-clone/backend/internal/adapters/redis"
	httpAdapter "github.com/wepink-clone/backend/internal/adapters/http"
	"github.com/wepink-clone/backend/internal/usecase"
)

func main() {
	// 1. Logging Setup (JSON)
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{AddSource: true})
	slog.SetDefault(slog.New(handler))

	slog.Info("Waiting for infrastructure to stabilize...")
	time.Sleep(5 * time.Second)

	// 2. Infrastructure Setup (Environment Variables)
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		dbURL = "user:pass@tcp(localhost:3306)/wepink?parseTime=true"
	}

	db, err := sql.Open("mysql", dbURL)
	if err != nil {
		slog.Error("Failed to open MySQL connection", "error", err)
	}

	// Test connection immediately
	if db != nil {
		if err := db.Ping(); err != nil {
			slog.Warn("MySQL not reachable yet", "error", err)
		}
	}

	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "localhost:6379"
	}

	rabbitURL := os.Getenv("RABBITMQ_URL")
	if rabbitURL == "" {
		rabbitURL = "amqp://guest:guest@localhost:5672/"
	}
	
	// MySQL Pool Configuration
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	redisClient := redis.NewClient(&redis.Options{
		Addr:         redisURL,
		PoolSize:     10,
		MinIdleConns: 5,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})
	
	rabbitAdapter, err := rabbitmq.NewRabbitMQAdapter(rabbitURL)
	if err != nil {
		slog.Warn("Failed to connect to RabbitMQ", "error", err)
	}

	// 3. Repositories & Adapters
	orderRepo := mysql.NewOrderRepository(db)
	paymentRepo := mysql.NewPaymentRepository(db)
	processedEventRepo := mysql.NewProcessedEventRepository(db)
	txManager := mysql.NewTransactionManager(db)
	
	idempotencyStore := redisAdapter.NewIdempotencyStore(redisClient)
	paymentGateway := payment.NewFakePaymentGateway()

	// 4. Use Cases
	orderUC := usecase.NewOrderUseCase(orderRepo, rabbitAdapter)
	paymentUC := usecase.NewPaymentUseCase(
		paymentRepo, 
		orderRepo, 
		idempotencyStore, 
		txManager, 
		rabbitAdapter, 
		paymentGateway,
	)

	// 5. Consumers
	if rabbitAdapter != nil {
		paymentConsumer := rabbitmq.NewPaymentConsumer(
			nil, // Channel initialization logic needed here
			paymentUC,
			processedEventRepo,
			txManager,
		)
		slog.Info("Payment consumer initialized")
		_ = paymentConsumer
	}

	// 6. HTTP Server & Middleware
	handlerHTTP := httpAdapter.NewOrderHandler(orderUC, paymentUC, db, rabbitAdapter)
	
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health/live", handlerHTTP.Live)
	mux.HandleFunc("GET /health/ready", handlerHTTP.Ready)
	mux.HandleFunc("POST /orders", handlerHTTP.CreateOrder)
	mux.HandleFunc("GET /orders/{id}", handlerHTTP.GetOrder)
	mux.HandleFunc("POST /payments/{orderId}", handlerHTTP.ProcessPayment)

	// Wrap mux with Correlation ID Middleware
	mainHandler := httpAdapter.CorrelationIDMiddleware(mux)

	srv := &http.Server{
		Addr:         ":8080",
		Handler:      mainHandler,
		WriteTimeout: 10 * time.Second,
		ReadTimeout:  10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	slog.Info("Server starting on :8080")
	if err := srv.ListenAndServe(); err != nil {
		slog.Error("Server failed", "error", err)
		os.Exit(1)
	}
}
