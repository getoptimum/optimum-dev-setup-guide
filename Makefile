P2P_CLIENT_DIR := ./grpc_p2p_client
PROXY_CLIENT_DIR := ./grpc_proxy_client
IDENTITY_DIR := ./identity

help: ## Show help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "Direct binary usage (recommended):"
	@echo "  # Subscribe to a topic"
	@echo "  $(P2P_CLIENT_DIR)/p2p-client -mode=subscribe -topic=\"testtopic\" --addr=\"127.0.0.1:33221\""
	@echo ""
	@echo "  # Publish messages"
	@echo "  $(P2P_CLIENT_DIR)/p2p-client -mode=publish -topic=\"testtopic\" --addr=\"127.0.0.1:33221\""
	@echo "  $(P2P_CLIENT_DIR)/p2p-client -mode=publish -topic=\"testtopic\" -msg=\"Hello World\" --addr=\"127.0.0.1:33221\""
	@echo ""
	@echo "  # Publish multiple messages with options"
	@echo "  $(P2P_CLIENT_DIR)/p2p-client -mode=publish -topic=\"testtopic\" --addr=\"127.0.0.1:33221\" -count=10 -sleep=1s"

build: ## Build all client binaries
	@echo "Building P2P client..."
	@cd $(P2P_CLIENT_DIR) && go build -o p2p-client ./p2p_client.go
	@echo "Building Proxy client..."
	@cd $(PROXY_CLIENT_DIR) && go build -o proxy-client ./proxy_client.go
	@echo "All clients built successfully!"

generate-identity: ## Generate P2P identity (if missing)
	@mkdir -p $(IDENTITY_DIR)
	@if [ ! -f "$(IDENTITY_DIR)/p2p.key" ]; then \
		echo "Generating new P2P identity..."; \
		cd keygen && go run .; \
	else \
		echo "P2P identity already exists at $(IDENTITY_DIR)/p2p.key"; \
	fi

clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	@rm -f $(P2P_CLIENT_DIR)/p2p-client
	@rm -f $(PROXY_CLIENT_DIR)/proxy-client
	@echo "Clean complete!"

.DEFAULT_GOAL := help
.PHONY: help build generate-identity clean
