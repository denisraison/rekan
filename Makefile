.PHONY: dev dev-api dev-web eval eval-judges

dev:
	$(MAKE) dev-api &
	@echo "Waiting for PocketBase on :8090..."
	@while ! nc -z localhost 8090 2>/dev/null; do sleep 0.2; done
	@echo "PocketBase ready"
	$(MAKE) dev-web

dev-api:
	cd api && go run .

dev-web:
	cd web && pnpm dev

eval:
	set -a && . ./.env && set +a && cd eval && go run ./cmd/eval

eval-judges:
	set -a && . ./.env && set +a && cd eval && go run ./cmd/eval --judges
