ROOT_DIR     := $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))
IMAGE_NAME   := topographer
APP_NAME     := topographer

GOLANGCI_LINT_VERSION := v1.12.3

MOD_ON  := env GO111MODULE=on
MOD_OFF := env GO111MODULE=auto

# -----------------------------------------------------------------------------

.PHONY: all docker clean lint test prepare install-lint

# -----------------------------------------------------------------------------


all: $(APP_NAME)

$(APP_NAME):
	@$(MOD_ON) go build -o "$(APP_NAME)" -ldflags="-s -w"

test:
	@$(MOD_ON) go test -v ./...

lint:
	@$(MOD_ON) golangci-lint run

clean:
	@git clean -xfd && \
		git reset --hard && \
		git submodule foreach --recursive sh -c 'git clean -xfd && git reset --hard'

docker:
	@docker build --pull -t "$(IMAGE_NAME)" "$(ROOT_DIR)"

prepare: install-lint

install-lint:
	@curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh \
		| $(MOD_OFF) bash -s -- -b $(GOPATH)/bin $(GOLANGCI_LINT_VERSION)
