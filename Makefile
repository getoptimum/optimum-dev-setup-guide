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

subscribe: build generate-identity ## subscribe to p2p topic: make subscribe <addr> <topic>
	@set -e; addr="$(word 2,$(MAKECMDGOALS))"; topic="$(word 3,$(MAKECMDGOALS))"; \
	if [ -z "$$addr" ] || [ -z "$$topic" ]; then \
		echo "Usage: make subscribe <addr> <topic>" >&2; \
		echo "Example: make subscribe 127.0.0.1:33221 testtopic" >&2; \
		exit 1; \
	fi; \
	$(P2P_CLIENT_DIR)/p2p-client -mode=subscribe -topic="$$topic" --addr="$$addr"

publish: build generate-identity ## publish message to p2p topic: make publish <addr> <topic> <message|random> [count] [sleep]
	@set -e; \
	addr="$(word 2,$(MAKECMDGOALS))"; \
	topic="$(word 3,$(MAKECMDGOALS))"; \
	message="$(word 4,$(MAKECMDGOALS))"; \
	count="$(word 5,$(MAKECMDGOALS))"; \
	sleep="$(word 6,$(MAKECMDGOALS))"; \
	if [ -z "$$addr" ] || [ -z "$$topic" ] || [ -z "$$message" ]; then \
		echo "Usage: make publish <addr> <topic> <message|random> [count] [sleep]" >&2; \
		echo "Examples:" >&2; \
		echo "  make publish 127.0.0.1:33221 testtopic random" >&2; \
		echo "  make publish 127.0.0.1:33221 testtopic random 10 1s" >&2; \
		exit 1; \
	fi; \
	extra_args=""; \
	if [ -n "$$count" ]; then extra_args="$$extra_args -count=$$count"; fi; \
	if [ -n "$$sleep" ]; then \
		case "$$sleep" in \
			*[0-9]) sleep_val="$$sleep"s ;; \
			*) sleep_val="$$sleep" ;; \
		esac; \
		extra_args="$$extra_args -sleep=$$sleep_val"; \
	fi; \
	if [ "$$message" = "random" ]; then \
		echo "Publishing random messages to topic=$$topic addr=$$addr count=$${count:-1} sleep=$${sleep:-default}"; \
		$(P2P_CLIENT_DIR)/p2p-client -mode=publish -topic="$$topic" -msg="random" --addr="$$addr" $$extra_args; \
	else \
		echo "Publishing message '$$message' to topic=$$topic addr=$$addr"; \
		$(P2P_CLIENT_DIR)/p2p-client -mode=publish -topic="$$topic" -msg="$$message" --addr="$$addr" $$extra_args; \
	fi

clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	@rm -f $(P2P_CLIENT_DIR)/p2p-client
	@rm -f $(PROXY_CLIENT_DIR)/proxy-client
	@echo "Clean complete!"

# Prevent make from interpreting arguments as targets
%:
	@:

.DEFAULT_GOAL := help
.PHONY: help build generate-identity subscribe publish clean
