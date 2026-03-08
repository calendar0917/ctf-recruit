.PHONY: backend-run backend-test frontend-install frontend-dev compose-up compose-down compose-logs prod-compose-up prod-compose-down prod-compose-logs bootstrap-admin

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

prod-compose-up:
	docker compose -f deploy/docker-compose.prod.yml up -d --build

prod-compose-down:
	docker compose -f deploy/docker-compose.prod.yml down

prod-compose-logs:
	docker compose -f deploy/docker-compose.prod.yml logs -f

bootstrap-admin:
	scripts/bootstrap-admin.sh
