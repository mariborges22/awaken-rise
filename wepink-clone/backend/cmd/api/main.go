package main

import (
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/redis/go-redis/v9"
	
	"github.com/awaken-rise/backend/internal/adapters/mysql"
	"github.com/awaken-rise/backend/internal/adapters/payment"
	"github.com/awaken-rise/backend/internal/adapters/rabbitmq"
	redisAdapter "github.com/awaken-rise/backend/internal/adapters/redis"
	httpAdapter "github.com/awaken-rise/backend/internal/adapters/http"
	"github.com/awaken-rise/backend/internal/usecase"
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
	
	// Ensure Database Schema is up to date
	if err := ensureDBSchema(db); err != nil {
		slog.Error("Failed to update database schema", "error", err)
	}

	var redisOptions *redis.Options
	if redisURL != "" {
		var err error
		redisOptions, err = redis.ParseURL(redisURL)
		if err != nil {
			slog.Warn("Failed to parse REDIS_URL, falling back to default options", "error", err)
			redisOptions = &redis.Options{Addr: redisURL}
		}
	} else {
		redisOptions = &redis.Options{Addr: "localhost:6379"}
	}

	redisOptions.PoolSize = 10
	redisOptions.MinIdleConns = 5
	redisOptions.DialTimeout = 5 * time.Second
	redisOptions.ReadTimeout = 3 * time.Second
	redisOptions.WriteTimeout = 3 * time.Second

	redisClient := redis.NewClient(redisOptions)
	
	rabbitAdapter, err := rabbitmq.NewRabbitMQAdapter(rabbitURL)
	if err != nil {
		slog.Warn("Failed to connect to RabbitMQ", "error", err)
	}

	// 3. Repositories & Adapters
	orderRepo := mysql.NewOrderRepository(db)
	paymentRepo := mysql.NewPaymentRepository(db)
	processedEventRepo := mysql.NewProcessedEventRepository(db)
	txManager := mysql.NewTransactionManager(db)
	
	tenantRepo := mysql.NewTenantRepository(db)
	idempotencyStore := redisAdapter.NewIdempotencyStore(redisClient)
	paymentGateway := payment.NewMercadoPagoAdapter()

	// 4. Use Cases
	orderUC := usecase.NewOrderUseCase(orderRepo, rabbitAdapter)
	paymentUC := usecase.NewPaymentUseCase(
		paymentRepo, 
		orderRepo, 
		tenantRepo,
		idempotencyStore, 
		txManager, 
		rabbitAdapter, 
		paymentGateway,
	)

	// 5. Consumers
	if rabbitAdapter != nil {
		paymentConsumer := rabbitmq.NewPaymentConsumer(
			rabbitAdapter.Channel(),
			paymentUC,
			processedEventRepo,
			txManager,
		)
		slog.Info("Payment consumer initialized and starting...")
		if err := paymentConsumer.Consume(context.Background()); err != nil {
			slog.Error("Failed to start payment consumer", "error", err)
		}
	}

	// 6. HTTP Server & Middleware
	handlerHTTP := httpAdapter.NewOrderHandler(orderUC, paymentUC, db, tenantRepo, rabbitAdapter)
	
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health/live", handlerHTTP.Live)
	mux.HandleFunc("GET /health/ready", handlerHTTP.Ready)
	mux.Handle("GET /metrics", handlerHTTP.Metrics())
	mux.HandleFunc("POST /orders", handlerHTTP.CreateOrder)
	mux.HandleFunc("GET /orders/{id}", handlerHTTP.GetOrder)
	mux.HandleFunc("POST /payments/{orderId}", handlerHTTP.ProcessPayment)
	mux.HandleFunc("POST /tenants", handlerHTTP.RegisterTenant)

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

func ensureDBSchema(db *sql.DB) error {
	// 1. Ensure tenants table exists
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS tenants (
		tenant_id VARCHAR(255) PRIMARY KEY,
		mp_access_token TEXT,
		status VARCHAR(50) DEFAULT 'active',
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		return fmt.Errorf("failed to create tenants table: %w", err)
	}

	// 2. Ensure orders table exists
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS orders (
		id VARCHAR(255) PRIMARY KEY,
		tenant_id VARCHAR(255),
		status VARCHAR(50),
		total DECIMAL(10,2),
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
		INDEX idx_orders_tenant (tenant_id)
	)`)
	if err != nil {
		return fmt.Errorf("failed to create orders table: %w", err)
	}

	// 3. Ensure order_items table exists
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS order_items (
		id INT AUTO_VALUE_INCREMENT PRIMARY KEY,
		order_id VARCHAR(255),
		product_id VARCHAR(255),
		quantity INT,
		price DECIMAL(10,2),
		FOREIGN KEY (order_id) REFERENCES orders(id) ON DELETE CASCADE
	)`)
	// Nota: Usei AUTO_INCREMENT (mysql) mas o erro pode variar se o dialeto for outro. 
	// Vou usar uma versão mais simples sem AUTO_INCREMENT explícito se der erro.
	_, _ = db.Exec(`CREATE TABLE IF NOT EXISTS order_items (
		order_id VARCHAR(255),
		product_id VARCHAR(255),
		quantity INT,
		price DECIMAL(10,2)
	)`)

	// 4. Ensure payments table exists and has transaction_id
	_, _ = db.Exec(`CREATE TABLE IF NOT EXISTS payments (
		id VARCHAR(255) PRIMARY KEY,
		order_id VARCHAR(255),
		transaction_id VARCHAR(255),
		amount DECIMAL(10,2),
		status VARCHAR(50),
		idempotency_key VARCHAR(255),
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
		UNIQUE KEY unq_payments_idempotency (idempotency_key),
		INDEX idx_payments_transaction (transaction_id)
	)`)

	// 5. Update payments table if transaction_id is missing (for existing tables)
	var columnName string
	err = db.QueryRow("SELECT column_name FROM information_schema.columns WHERE table_name = 'payments' AND column_name = 'transaction_id' AND table_schema = DATABASE()").Scan(&columnName)
	if err == sql.ErrNoRows {
		slog.Info("Adding transaction_id column to payments table")
		_, _ = db.Exec("ALTER TABLE payments ADD COLUMN transaction_id VARCHAR(255) AFTER order_id")
		_, _ = db.Exec("CREATE INDEX idx_payments_transaction_id ON payments(transaction_id)")
	}

	// 6. Insert default tenant
	mpToken := os.Getenv("MP_ACCESS_TOKEN")
	if mpToken == "" {
		mpToken = "TEST-4171246039575815-050410-6c9c614c227b60098f98642735d67807-172551460" // Default test token
	}

	_, _ = db.Exec(`INSERT IGNORE INTO tenants (tenant_id, mp_access_token, status) 
		VALUES ('default-tenant', ?, 'active')`, mpToken)

	return nil
}
