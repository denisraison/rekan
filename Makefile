.PHONY: dev dev-mock dev-api dev-web eval eval-judges eval-fast test-judges lint seed deploy

dev:
	$(MAKE) dev-api &
	@echo "Waiting for PocketBase on :8090..."
	@while ! nc -z localhost 8090 2>/dev/null; do sleep 0.2; done
	@echo "PocketBase ready"
	$(MAKE) dev-web

dev-mock:
	$(MAKE) dev-api &
	@echo "Waiting for PocketBase on :8090..."
	@while ! nc -z localhost 8090 2>/dev/null; do sleep 0.2; done
	@echo "PocketBase ready"
	cd web && pnpm dev:mock --host 0.0.0.0

dev-api:
	set -a && . ./.env && set +a && cd api && go run . serve --http=0.0.0.0:8090

dev-web:
	cd web && pnpm dev --host 0.0.0.0

eval:
	set -a && . ./.env && set +a && cd eval && go run ./cmd/eval

eval-judges:
	set -a && . ./.env && set +a && cd eval && go run ./cmd/eval --judges

eval-fast:
	set -a && . ./.env && set +a && cd eval && go run ./cmd/eval --fast

test-judges:
	set -a && . ./.env && set +a && cd eval && go test -tags integration -v -run TestJudge

lint:
	cd api && golangci-lint run ./...

seed:
	set -a && . ./.env && set +a && bash scripts/seed.sh

INFRA_DIR    := ../infra
SERVER       := root@46.225.161.186

deploy: ## Tag, cross-build locally, push to server, activate (usage: make deploy [TAG=v0.4.3])
ifndef TAG
	$(eval TAG := $(shell git tag --sort=-v:refname | head -1 | awk -F. '{$$NF=$$NF+1; print}' OFS=.))
endif
	@test -n "$(TAG)" || { echo "ERROR: could not determine tag"; exit 1; }
	@echo "Deploying $(TAG)"
	git tag $(TAG) && git push origin $(TAG)
	cd $(INFRA_DIR) && nix flake lock --override-input rekan github:denisraison/rekan/$(TAG)
	cd $(INFRA_DIR) && nix build .#nixosConfigurations.prod.config.services.rekan.instances.prod.package -o result-api & \
	cd $(INFRA_DIR) && nix build .#nixosConfigurations.prod.config.services.rekan.instances.prod.webRoot -o result-web & \
	wait
	nix copy --to ssh://$(SERVER) $(INFRA_DIR)/result-api $(INFRA_DIR)/result-web
	cd $(INFRA_DIR) && git add flake.lock && git commit -m "Update rekan to $(TAG)" && git push
	ssh $(SERVER) "nixos-rebuild switch --flake github:denisraison/infra#prod --refresh"
