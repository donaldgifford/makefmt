# Project Variables

PROJECT_NAME := my-project
PROJECT_OWNER := donaldgifford
DESCRIPTION := A project

## Go Variables

GO ?= go
GO_PACKAGE := github.com/$(PROJECT_OWNER)/$(PROJECT_NAME)

###############
##@ Development

.PHONY: build test lint

build: ## Build the binary
	@ $(MAKE) --no-print-directory log-$@
	@mkdir -p $(BIN_DIR)
	@go build -o $(BIN_DIR)/$(PROJECT_NAME) ./cmd/$(PROJECT_NAME)

test: ## Run tests
	@ $(MAKE) --no-print-directory log-$@
	@go test -v -race ./...


release: ## Create release (use with TAG=v1.0.0)
	@ $(MAKE) --no-print-directory log-$@
	@if [ -z "$(TAG)" ]; then                                                    \
		echo "Error: TAG is required";                                              \
			exit 1;                                                                    \
	fi
	git tag -a $(TAG) -m "Release $(TAG)"


########
##@ Help

log-%:
	@grep -h -E '^$*:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN { FS = ":.*?## " }; { printf "\033[36m==> %s\033[0m\n", $$2 }'
