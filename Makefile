IDENTITY_DIR="./identity"
KEY_FILE="$IDENTITY_DIR/p2p.key"
P2P_CLIENT_DIR := ./grpc_p2p_client
P2P_CLIENT_BIN := ./grpc_p2p_client/p2p-client

help: ## Show help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

generate-key: ## Generate p2p identity (if missing)
	@mkdir -p $(IDENTITY_DIR)
	@echo "Generating new p2p identity if not exist..."; cd keygen && go run .

build-client: ## Build executable p2p client
	@echo "Generating p2p client code..."; cd $(P2P_CLIENT_DIR) && go build -o p2p-client ./p2p_client.go

subscribe: build-client generate-key ## subscribe to p2p topic
	@set -e; addr="$(word 2,$(MAKECMDGOALS))"; topic="$(word 3,$(MAKECMDGOALS))"; \
	if [ -z "$$addr" ] || [ -z "$$topic" ]; then \
		echo "Usage: make subscribe <addr> <topic>" >&2; \
		exit 1; \
	fi; \
	$(P2P_CLIENT_BIN) -mode=subscribe -topic="$$topic" --addr="$$addr"

publish: build-client generate-key ## publish message to p2p topic
	@set -e; \
	addr="$(word 2,$(MAKECMDGOALS))"; \
	topic="$(word 3,$(MAKECMDGOALS))"; \
	message="$(word 4,$(MAKECMDGOALS))"; \
	if [ -z "$$addr" ] || [ -z "$$topic" ] || [ -z "$$message" ]; then \
		echo "Usage: make publish <addr> <topic> <message|random> [count=N] [sleep=Xs]" >&2; \
		exit 1; \
	fi; \
	shifted=$$(echo $(MAKECMDGOALS) | cut -d' ' -f5-); \
	if [ "$$message" = "random" ]; then \
		echo "Publishing random messages to topic=$$topic addr=$$addr"; \
		$(P2P_CLIENT_BIN) -mode=publish -topic="$$topic" --addr="$$addr" $$shifted; \
	else \
		echo "Publishing message '$$message' to topic=$$topic addr=$$addr"; \
		$(P2P_CLIENT_BIN) -mode=publish -topic="$$topic" -msg="$$message" --addr="$$addr" $$shifted; \
	fi

.DEFAULT_GOAL := help
.PHONY: help generate-key build-client subscribe publish
