package app

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/redis/go-redis/v9"
	
	httpAdapter "github.com/awaken-rise/backend/internal/adapters/http"
	"github.com/awaken-rise/backend/internal/adapters/mysql"
	"github.com/awaken-rise/backend/internal/adapters/payment"
	"github.com/awaken-rise/backend/internal/adapters/rabbitmq"
	redisAdapter "github.com/awaken-rise/backend/internal/adapters/redis"
	"github.com/awaken-rise/backend/internal/database"
	"github.com/awaken-rise/backend/internal/domain/service"
	"github.com/awaken-rise/backend/internal/pkg/events"
	"github.com/awaken-rise/backend/internal/pkg/workers"
	"github.com/awaken-rise/backend/internal/usecase"
)

type App struct {
	httpServer *http.Server
	db         *sql.DB
	redis      *redis.Client
	rabbit     *rabbitmq.RabbitMQAdapter
}

func NewApp() *App {
	return &App{}
}

func (a *App) Start() error {
	ctx := context.Background()

	// 1. Database Connection
	db, err := sql.Open("mysql", os.Getenv("DB_URL"))
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Configuração do Connection Pool para Escalabilidade
	db.SetMaxOpenConns(100)          // Máximo de conexões abertas simultaneamente (evita sobrecarga no MySQL)
	db.SetMaxIdleConns(10)           // Máximo de conexões ociosas mantidas abertas
	db.SetConnMaxLifetime(time.Hour) // Tempo máximo de vida de uma conexão (evita conexões "presas" ou stale)
	db.SetConnMaxIdleTime(10 * time.Minute) // Fecha conexões ociosas após 10 min

	a.db = db

	// 2. Run Migrations
	migrationsPath := os.Getenv("MIGRATIONS_PATH")
	if migrationsPath == "" {
		migrationsPath = "database/migrations"
	}
	if err := database.RunMigrations(db, migrationsPath); err != nil {
		slog.Warn("Migrations failed or no changes", "error", err)
	}

	// 3. Infrastructure (Redis, RabbitMQ)
	redisURL := os.Getenv("REDIS_URL")
	redisOptions, err := redis.ParseURL(redisURL)
	if err != nil {
		slog.Warn("Failed to parse REDIS_URL, using default options", "error", err)
		redisOptions = &redis.Options{Addr: "localhost:6379"}
	}
	a.redis = redis.NewClient(redisOptions)

	rabbitAdapter, err := rabbitmq.NewRabbitMQAdapter(os.Getenv("RABBITMQ_URL"))
	if err != nil {
		slog.Warn("Failed to connect to RabbitMQ", "error", err)
	} else {
		// Garantir que a fila principal e a DLQ existam com a configuração correta
		if err := rabbitAdapter.DeclareResilientQueue("payment_queue"); err != nil {
			slog.Error("Failed to declare resilient queue", "error", err)
		}
		a.rabbit = rabbitAdapter
	}

	// 4. Repositories
	orderRepo := mysql.NewOrderRepository(db)
	paymentRepo := mysql.NewPaymentRepository(db)
	processedEventRepo := mysql.NewProcessedEventRepository(db)
	txManager := mysql.NewTransactionManager(db)
	tenantRepo := mysql.NewTenantRepository(db)
	userRepo := mysql.NewUserRepository(db)
	productRepo := mysql.NewProductRepository(db)
	idempotencyStore := redisAdapter.NewIdempotencyStore(a.redis)
	productCache := redisAdapter.NewProductCacheStore(a.redis)
	paymentGateway := payment.NewMercadoPagoAdapter()
	idempotencyService := service.NewIdempotencyService(idempotencyStore, paymentRepo)

	// Encryption Key (Must be 32 bytes for AES-256)
	encKey := os.Getenv("ENCRYPTION_KEY")
	if encKey == "" {
		encKey = "awaken-rise-secret-key-32-bytes!" 
	}
	encryptionService, _ := service.NewEncryptionService(encKey)

	outboxRepo := mysql.NewOutboxRepository(db)
	eventDispatcher := events.NewOutboxEventDispatcher(outboxRepo)

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "awaken-rise-jwt-secret-key-32-bytes!"
	}
	authUC := usecase.NewAuthUseCase(userRepo, jwtSecret)
	productUC := usecase.NewProductUseCase(productRepo, productCache)

	// 5. Use Cases
	orderUC := usecase.NewOrderUseCase(orderRepo, productRepo, eventDispatcher, txManager)
	tenantOnboardingUC := usecase.NewTenantOnboardingUseCase(tenantRepo)
	paymentUC := usecase.NewPaymentUseCase(
		paymentRepo, 
		orderRepo, 
		tenantRepo,
		txManager, 
		eventDispatcher, 
		paymentGateway,
		paymentGateway, // OAuthProvider: MercadoPagoAdapter implementa ambas as interfaces
		idempotencyService,
		encryptionService,
	)

	// 6. Consumers and Workers
	if rabbitAdapter != nil {
		relayWorker := workers.NewOutboxRelayWorker(outboxRepo, rabbitAdapter, 5*time.Second, 100)
		go relayWorker.Start(ctx)

		paymentConsumer := rabbitmq.NewPaymentConsumer(
			rabbitAdapter.Channel(),
			paymentUC,
			processedEventRepo,
			txManager,
		)
		go func() {
			slog.Info("Payment consumer starting...")
			if err := paymentConsumer.Consume(ctx); err != nil {
				slog.Error("Payment consumer failed", "error", err)
			}
		}()
	}

	// 7. HTTP Server
	handlerHTTP := httpAdapter.NewOrderHandler(orderUC, paymentUC, db, tenantRepo, eventDispatcher, tenantOnboardingUC)
	handlerHTTP.SetEncryptionService(encryptionService)
	handlerHTTP.SetAuthUseCase(authUC)
	handlerHTTP.SetProductUseCase(productUC)

	billingUC := usecase.NewBillingUseCase(tenantRepo)
	handlerHTTP.SetBillingUseCase(billingUC)
	handlerHTTP.SetMPOAuthProvider(paymentGateway) // Injeta o adapter OAuth

	authMiddleware := httpAdapter.NewAuthMiddleware(jwtSecret)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health/live", handlerHTTP.Live)
	mux.HandleFunc("GET /health/ready", handlerHTTP.Ready)
	mux.Handle("GET /metrics", handlerHTTP.Metrics())
	mux.HandleFunc("POST /orders", handlerHTTP.CreateOrder)
	mux.HandleFunc("GET /orders/{id}", handlerHTTP.GetOrder)
	mux.HandleFunc("POST /payments/{orderId}", handlerHTTP.ProcessPayment)
	mux.HandleFunc("POST /tenants", handlerHTTP.RegisterTenant)
	mux.HandleFunc("POST /auth/register", handlerHTTP.RegisterUser)
	mux.HandleFunc("POST /auth/login", handlerHTTP.Login)
	mux.HandleFunc("POST /webhooks/billing", handlerHTTP.BillingWebhook)
	mux.Handle("GET /tenants/me/config", authMiddleware.Handler(http.HandlerFunc(handlerHTTP.GetTenantConfig)))
	mux.Handle("PUT /tenants/me/config", authMiddleware.Handler(http.HandlerFunc(handlerHTTP.UpdateTenantConfig)))
	mux.Handle("POST /products", authMiddleware.Handler(http.HandlerFunc(handlerHTTP.CreateProduct)))
	mux.HandleFunc("GET /products", handlerHTTP.ListProducts)
	// Rotas OAuth Mercado Pago
	mux.Handle("GET /auth/mercadopago/url", authMiddleware.Handler(http.HandlerFunc(handlerHTTP.MercadoPagoOAuthURL)))
	mux.HandleFunc("GET /auth/mercadopago/callback", handlerHTTP.MercadoPagoOAuthCallback)

	a.httpServer = &http.Server{
		Addr: ":8080",
		Handler: httpAdapter.CorrelationIDMiddleware(
			httpAdapter.TenantIDMiddleware(
				httpAdapter.MetricsMiddleware(db)(mux),
			),
		),
	}

	// Graceful Shutdown
	go a.listenForShutdown()

	slog.Info("Server starting on :8080")
	return a.httpServer.ListenAndServe()
}

func (a *App) listenForShutdown() {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	slog.Info("Shutting down gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if a.httpServer != nil {
		a.httpServer.Shutdown(ctx)
	}
	if a.db != nil {
		a.db.Close()
	}
	if a.rabbit != nil {
		a.rabbit.Close()
	}
	slog.Info("Shutdown complete")
}
