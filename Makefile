P2P_CLIENT_DIR := ./grpc_p2p_client
PROXY_CLIENT_DIR := ./grpc_proxy_client
IDENTITY_DIR := ./identity

# Build targets
P2P_CLIENT := $(P2P_CLIENT_DIR)/p2p-client
PROXY_CLIENT := $(PROXY_CLIENT_DIR)/proxy-client
KEYGEN_BINARY := keygen/generate-p2p-key
DASHBOARD_BINARY := tools/network-dashboard/network-dashboard

# Scripts
SCRIPTS := ./script/generate-identity.sh ./script/proxy_client.sh ./test_suite.sh

# Helper targets (not shown in help)
.PHONY: $(P2P_CLIENT) $(PROXY_CLIENT) $(KEYGEN_BINARY) $(DASHBOARD_BINARY) setup-scripts

$(P2P_CLIENT):
	@cd $(P2P_CLIENT_DIR) && go build -o p2p-client ./cmd/single/
	@cd $(P2P_CLIENT_DIR) && go build -o p2p-multi-publish ./cmd/multi-publish/
	@cd $(P2P_CLIENT_DIR) && go build -o p2p-multi-subscribe ./cmd/multi-subscribe/

$(PROXY_CLIENT):
	@cd $(PROXY_CLIENT_DIR) && go build -o proxy-client ./proxy_client.go

$(KEYGEN_BINARY):
	@cd keygen && go build -o generate-p2p-key ./generate_p2p_key.go

$(DASHBOARD_BINARY):
	@cd tools/network-dashboard && go build -o network-dashboard .

setup-scripts:
	@chmod +x $(SCRIPTS)

help: ## Show help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "Direct binary usage (recommended):"
	@echo "  # Subscribe to a topic"
	@echo "  $(P2P_CLIENT) -mode=subscribe -topic=\"testtopic\" --addr=\"127.0.0.1:33221\""
	@echo ""
	@echo "  # Publish messages"
	@echo "  $(P2P_CLIENT) -mode=publish -topic=\"testtopic\" -msg=\"Hello World\" --addr=\"127.0.0.1:33221\""
	@echo "  $(P2P_CLIENT) -mode=publish -topic=\"testtopic\" -msg=\"Random Message\" --addr=\"127.0.0.1:33221\""
	@echo ""
	@echo "  # Publish multiple messages with options"
	@echo "  $(P2P_CLIENT) -mode=publish -topic=\"testtopic\" -msg=\"Random Message\" --addr=\"127.0.0.1:33221\" -count=10 -sleep=1s"

build: $(P2P_CLIENT) $(PROXY_CLIENT) $(DASHBOARD_BINARY) ## Build all client binaries

generate-identity: ## Generate P2P identity (if missing)
	@mkdir -p $(IDENTITY_DIR)
	@if [ ! -f "$(IDENTITY_DIR)/p2p.key" ]; then \
		echo "Generating new P2P identity..."; \
		cd keygen && go run .; \
	else \
		echo "P2P identity already exists at $(IDENTITY_DIR)/p2p.key"; \
	fi

subscribe: $(P2P_CLIENT) generate-identity ## subscribe to p2p topic: make subscribe <addr> <topic>
	@set -e; addr="$(word 2,$(MAKECMDGOALS))"; topic="$(word 3,$(MAKECMDGOALS))"; \
	if [ -z "$$addr" ] || [ -z "$$topic" ]; then \
		echo "Usage: make subscribe <addr> <topic>" >&2; \
		echo "Example: make subscribe 127.0.0.1:33221 testtopic" >&2; \
		exit 1; \
	fi; \
	$(P2P_CLIENT) -mode=subscribe -topic="$$topic" --addr="$$addr"

publish: $(P2P_CLIENT) generate-identity ## publish message to p2p topic: make publish <addr> <topic> <message|random> [count] [sleep]
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
		$(P2P_CLIENT) -mode=publish -topic="$$topic" -msg="random" --addr="$$addr" $$extra_args; \
	else \
		$(P2P_CLIENT) -mode=publish -topic="$$topic" -msg="$$message" --addr="$$addr" $$extra_args; \
	fi

test: $(P2P_CLIENT) $(PROXY_CLIENT) $(KEYGEN_BINARY) ## Run tests for Go clients

lint: ## Run golangci-lint
	@cd $(P2P_CLIENT_DIR) && golangci-lint run --skip-dirs-use-default || echo "Linting issues found in P2P client"
	@cd $(PROXY_CLIENT_DIR) && golangci-lint run --skip-dirs-use-default || echo "Linting issues found in Proxy client"
	@cd keygen && golangci-lint run --skip-dirs-use-default || echo "Linting issues found in Keygen"

test-docker: setup-scripts ## Test Docker Compose setup
	@./script/generate-identity.sh
	@docker-compose -f docker-compose-optimum.yml up --build -d
	# Alternative: Use GossipSub protocol instead
	# @docker-compose -f docker-compose-gossipsub.yml up --build -d
	@sleep 30
	@docker-compose -f docker-compose-optimum.yml ps
	@./test_suite.sh

test-scripts: setup-scripts ## Test shell scripts

validate: ## Validate configuration files
	@docker-compose -f docker-compose-optimum.yml config
	# Alternative: Validate GossipSub configuration instead
	# @docker-compose -f docker-compose-gossipsub.yml config
	@cd $(P2P_CLIENT_DIR) && go mod verify
	@cd $(PROXY_CLIENT_DIR) && go mod verify
	@cd keygen && go mod verify

ci: test lint test-docker test-scripts validate ## Run all CI checks locally

dashboard: $(DASHBOARD_BINARY) ## Show network health dashboard: make dashboard [local|remote] [proxy-base=URL] [node-base=URL]
	@set -e; \
	mode="$(word 2,$(MAKECMDGOALS))"; \
	proxy_base="$(proxy-base)"; \
	node_base="$(node-base)"; \
	if [ -z "$$mode" ] || [ "$$mode" = "local" ]; then \
		$(DASHBOARD_BINARY) -local; \
	elif [ "$$mode" = "remote" ]; then \
		args=""; \
		if [ -n "$$proxy_base" ]; then args="$$args -proxy-base=$$proxy_base"; fi; \
		if [ -n "$$node_base" ]; then args="$$args -node-base=$$node_base"; fi; \
		if [ -z "$$args" ]; then \
			exit 1; \
		fi; \
		$(DASHBOARD_BINARY) $$args; \
	else \
		exit 1; \
	fi

clean: ## Clean build artifacts
	@rm -f $(P2P_CLIENT) $(PROXY_CLIENT) $(KEYGEN_BINARY) $(DASHBOARD_BINARY)

# Prevent make from interpreting arguments as targets
%:
	@:

.DEFAULT_GOAL := help
.PHONY: help build generate-identity subscribe publish test lint test-docker test-scripts validate ci clean setup-scripts dashboard
