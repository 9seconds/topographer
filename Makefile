ROOT_DIR     := $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))
IMAGE_NAME   := topographer
APP_NAME     := topographer
GOMETALINTER := gometalinter.v2

# -----------------------------------------------------------------------------

.PHONY: all docker clean lint test install_cli install_dep install_lint

# -----------------------------------------------------------------------------


all: $(APP_NAME)

$(APP_NAME): vendor
	@go build -o "$(APP_NAME)" -ldflags="-s -w"

vendor: Gopkg.lock Gopkg.toml install_cli
	@dep ensure

test: install_cli
	@go test -v ./...

lint: vendor install_cli
	@$(GOMETALINTER) --deadline=2m ./...

clean:
	@git clean -xfd && \
		git reset --hard && \
		git submodule foreach --recursive sh -c 'git clean -xfd && git reset --hard' && \
		rm -rf ./vendor

docker:
	@docker build --pull -t "$(IMAGE_NAME)" "$(ROOT_DIR)"

install_cli: install_dep install_lint

install_dep:
	@go get github.com/golang/dep/cmd/dep

install_lint:
	@go get gopkg.in/alecthomas/gometalinter.v2 && \
		$(GOMETALINTER) --install
