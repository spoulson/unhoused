GOLANGCI_LINT := bin/golangci-lint

.PHONY: lint

lint: $(GOLANGCI_LINT)
	$(GOLANGCI_LINT) run ./...

$(GOLANGCI_LINT):
	./scripts/install-golangci-lint.sh
