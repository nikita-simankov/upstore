.PHONY: dev-db dev-api dev-admin dev-storefront tidy

dev-db:
	docker compose up -d postgres minio

dev-api: dev-db
	cd backend && go run ./cmd/api

dev-admin:
	cd frontend/admin && npm run dev

dev-storefront:
	cd frontend/storefront && npm run dev

tidy:
	cd backend && go mod tidy
