.PHONY: all build run test test-unit test-integration lint clean verify hooks

AGENT_DIR=packages/agent

all: build

build:
	@$(MAKE) -C $(AGENT_DIR) build

dev:
	@$(MAKE) -C $(AGENT_DIR) dev

run:
	@$(MAKE) -C $(AGENT_DIR) run

test:
	@$(MAKE) -C $(AGENT_DIR) test

test-unit:
	@$(MAKE) -C $(AGENT_DIR) test-unit

test-integration:
	@$(MAKE) -C $(AGENT_DIR) test-integration

lint:
	@$(MAKE) -C $(AGENT_DIR) lint

verify:
	@$(MAKE) -C $(AGENT_DIR) verify

hooks:
	@$(MAKE) -C $(AGENT_DIR) hooks

clean:
	@$(MAKE) -C $(AGENT_DIR) clean
