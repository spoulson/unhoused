GOLANGCI_LINT := bin/golangci-lint

.PHONY: lint
lint: $(GOLANGCI_LINT)
	$(GOLANGCI_LINT) run ./...

$(GOLANGCI_LINT):
	./scripts/install-golangci-lint.sh

.PHONY: dev
dev:
	docker compose -f docker-compose.yaml -f docker-compose.dev.yaml up --build
