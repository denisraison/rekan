.PHONY: dev dev-api dev-web

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
