.PHONY: dev-db dev-api dev-admin tidy

dev-db:
	docker compose up -d postgres

dev-api: dev-db
	cd backend && go run ./cmd/api

dev-admin:
	cd frontend/admin && npm run dev

tidy:
	cd backend && go mod tidy
