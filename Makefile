ROOT_DIR     := $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))
IMAGE_NAME   := topographer
APP_NAME     := topographer
GOMETALINTER := gometalinter.v2


all: build_prod
alldeps: deps devdeps
deps: deps_dep
devdeps: deps_gometalinter


build_dev:
	@cd $(ROOT_DIR) && go build -o "$(APP_NAME)"

build_prod: dep_ensure
	@cd $(ROOT_DIR) && go build -o "$(APP_NAME)" -ldflags="-s -w"

deps_dep:
	@go get -u github.com/golang/dep/cmd/dep

deps_gometalinter:
	@go get -u gopkg.in/alecthomas/gometalinter.v2 \
		&& $(GOMETALINTER) --install

lint:
	@cd $(ROOT_DIR) && $(GOMETALINTER) ./...

lint_dep: deps_gometalinter
	@cd $(ROOT_DIR) && $(GOMETALINTER) ./...

test:
	@cd $(ROOT_DIR) && go test -v ./...

dep_ensure:
	@cd $(ROOT_DIR) && dep ensure

clean:
	@cd $(ROOT_DIR) && git clean -xfd && git reset --hard \
		&& git submodule foreach --recursive sh -c 'git clean -xfd && git reset --hard'

docker:
	@docker build --pull -t $(IMAGE_NAME) $(ROOT_DIR)
