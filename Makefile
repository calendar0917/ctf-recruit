.PHONY: backend-run backend-test frontend-install frontend-dev compose-up compose-down compose-logs

backend-run:
	cd backend && go run ./cmd/api

backend-test:
	cd backend && go test ./...

frontend-install:
	cd frontend && pnpm install

frontend-dev:
	cd frontend && pnpm dev

compose-up:
	docker compose -f deploy/docker-compose.yml up --build

compose-down:
	docker compose -f deploy/docker-compose.yml down

compose-logs:
	docker compose -f deploy/docker-compose.yml logs -f
