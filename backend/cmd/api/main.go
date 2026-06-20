package main

import (
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/joho/godotenv"
	"github.com/nikita-simankov/upstore/internal/analytics"
	"github.com/nikita-simankov/upstore/internal/auth"
	"github.com/nikita-simankov/upstore/internal/billing"
	"github.com/nikita-simankov/upstore/internal/checkout"
	"github.com/nikita-simankov/upstore/internal/db"
	"github.com/nikita-simankov/upstore/internal/order"
	"github.com/nikita-simankov/upstore/internal/payment"
	"github.com/nikita-simankov/upstore/internal/product"
	"github.com/nikita-simankov/upstore/internal/shipping"
	"github.com/nikita-simankov/upstore/internal/store"
	"github.com/nikita-simankov/upstore/internal/upload"
	"github.com/nikita-simankov/upstore/internal/webhook"
)

func main() {
	_ = godotenv.Load()

	database, err := db.Connect(os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer database.Close()

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET is required")
	}

	authSvc := auth.NewService(database, jwtSecret)
	authHandler := auth.NewHandler(authSvc)

	storeRepo := store.NewRepository(database)
	storeHandler := store.NewHandler(storeRepo)

	productRepo := product.NewRepository(database)
	productHandler := product.NewHandler(productRepo, storeRepo)

	orderRepo := order.NewRepository(database)
	orderHandler := order.NewHandler(orderRepo, storeRepo)

	analyticsHandler := analytics.NewHandler(database, storeRepo)
	billingMw := billing.NewMiddleware(database, storeRepo)

	var yukassaClient *payment.YukassaClient
	if shopID := os.Getenv("YUKASSA_SHOP_ID"); shopID != "" {
		ykClient := payment.NewYukassaClient(shopID, os.Getenv("YUKASSA_SECRET_KEY"))
		yukassaClient = ykClient
	}

	storefrontURL := getEnv("STOREFRONT_URL", "http://localhost:3000")
	checkoutHandler := checkout.NewHandler(storeRepo, productRepo, orderRepo, yukassaClient, storefrontURL)
	webhookHandler := webhook.NewHandler(orderRepo)

	var cdekClient *shipping.CDEKClient
	if cdekID := os.Getenv("CDEK_CLIENT_ID"); cdekID != "" {
		cdekClient = shipping.NewCDEKClient(cdekID, os.Getenv("CDEK_CLIENT_SECRET"))
	}
	shippingHandler := shipping.NewHandler(cdekClient)

	allowedOrigins := []string{getEnv("CORS_ORIGIN", "http://localhost:5173"), storefrontURL}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: allowedOrigins,
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Accept", "Authorization", "Content-Type"},
	}))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })

	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/auth/register", authHandler.Register)
		r.Post("/auth/login", authHandler.Login)

		r.Get("/public/stores/{slug}", storeHandler.GetPublic)
		r.Get("/public/stores/{slug}/products", productHandler.ListPublic)
		r.Get("/public/stores/{slug}/products/{id}", productHandler.GetPublic)
		r.Post("/public/stores/{slug}/checkout", checkoutHandler.Checkout)
		r.Get("/public/orders/{orderId}", checkoutHandler.GetOrder)
		r.Get("/public/shipping/rates", shippingHandler.Rates)

		r.Post("/webhooks/yukassa", webhookHandler.Yukassa)

		r.Group(func(r chi.Router) {
			r.Use(auth.Middleware(jwtSecret))

			r.Get("/me", authHandler.Me)

			r.Post("/stores", storeHandler.Create)
			r.Get("/stores", storeHandler.List)
			r.Get("/stores/{id}", storeHandler.Get)
			r.Put("/stores/{id}", storeHandler.Update)

			r.Route("/stores/{storeId}/products", func(r chi.Router) {
				r.With(billingMw.EnforceProductLimit).Post("/", productHandler.Create)
				r.Get("/", productHandler.List)
				r.Get("/{id}", productHandler.Get)
				r.Put("/{id}", productHandler.Update)
				r.Delete("/{id}", productHandler.Delete)
			})

			r.Get("/stores/{storeId}/orders", orderHandler.List)
			r.Put("/stores/{storeId}/orders/{id}/status", orderHandler.UpdateStatus)
			r.Get("/stores/{storeId}/analytics", analyticsHandler.Stats)

			if s3Endpoint := os.Getenv("S3_ENDPOINT"); s3Endpoint != "" {
				uploadHandler, err := upload.NewHandler(
					s3Endpoint,
					os.Getenv("S3_KEY_ID"),
					os.Getenv("S3_SECRET_KEY"),
					os.Getenv("S3_BUCKET"),
					os.Getenv("S3_PUBLIC_URL"),
				)
				if err != nil {
					log.Printf("upload handler init failed: %v", err)
				} else {
					r.Get("/upload/presign", uploadHandler.Presign)
				}
			}
		})
	})

	addr := ":" + getEnv("PORT", "8080")
	log.Printf("listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, r))
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
