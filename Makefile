BUILDDIR=./build
PLUGINDIR=./plugins
GOTIFY_VERSION=v2.6.1
PLUGIN_NAME=gotify-to-telegram
PLUGIN_ENTRY=plugin.go
GO_VERSION=`cat $(BUILDDIR)/gotify-server-go-version`
DOCKER_BUILD_IMAGE=gotify/build
DOCKER_WORKDIR=/proj
DOCKER_RUN=docker run --rm -v "$$PWD/.:${DOCKER_WORKDIR}" -v "`go env GOPATH`/pkg/mod/.:/go/pkg/mod:ro" -w ${DOCKER_WORKDIR}
DOCKER_GO_BUILD=go build -mod=readonly -a -installsuffix cgo -ldflags "$$LD_FLAGS" -buildmode=plugin 
GOMOD_CAP=go run github.com/gotify/plugin-api/cmd/gomod-cap

download-tools:
	GO111MODULE=off go get -u github.com/gotify/plugin-api/cmd/gomod-cap

create-build-dir:
	mkdir -p ${BUILDDIR} || true

update-go-mod: create-build-dir
	wget -LO ${BUILDDIR}/gotify-server.mod https://raw.githubusercontent.com/gotify/server/${GOTIFY_VERSION}/go.mod
	$(GOMOD_CAP) -from ${BUILDDIR}/gotify-server.mod -to go.mod
	rm ${BUILDDIR}/gotify-server.mod || true
	go mod tidy

get-gotify-server-go-version: create-build-dir
	rm ${BUILDDIR}/gotify-server-go-version || true
	wget -LO ${BUILDDIR}/gotify-server-go-version https://raw.githubusercontent.com/gotify/server/${GOTIFY_VERSION}/GO_VERSION

build-linux-amd64: get-gotify-server-go-version update-go-mod
	${DOCKER_RUN} ${DOCKER_BUILD_IMAGE}:$(GO_VERSION)-linux-amd64 ${DOCKER_GO_BUILD} -o ${BUILDDIR}/${PLUGIN_NAME}-linux-amd64${FILE_SUFFIX}.so ${DOCKER_WORKDIR}

build-linux-arm-7: get-gotify-server-go-version update-go-mod
	${DOCKER_RUN} ${DOCKER_BUILD_IMAGE}:$(GO_VERSION)-linux-arm-7 ${DOCKER_GO_BUILD} -o ${BUILDDIR}/${PLUGIN_NAME}-linux-arm-7${FILE_SUFFIX}.so ${DOCKER_WORKDIR}

build-linux-arm64: get-gotify-server-go-version update-go-mod
	${DOCKER_RUN} ${DOCKER_BUILD_IMAGE}:$(GO_VERSION)-linux-arm64 ${DOCKER_GO_BUILD} -o ${BUILDDIR}/${PLUGIN_NAME}-linux-arm64${FILE_SUFFIX}.so ${DOCKER_WORKDIR}

build: build-linux-arm-7 build-linux-amd64 build-linux-arm64

check-env:
	@if [ ! -f .env ]; then \
		echo "Creating .env from .example.env..."; \
		cp .example.env .env; \
	fi

compose-up: check-env
	docker compose up -d

compose-down:
	docker compose down --volumes

test:
	go test -v ./...

create-plugin-dir:
	mkdir -p ${PLUGINDIR}

move-plugin-arm64: create-plugin-dir build-linux-arm64
	cp ${BUILDDIR}/${PLUGIN_NAME}-linux-arm64${FILE_SUFFIX}.so ${PLUGINDIR}

move-plugin-amd64: create-plugin-dir build-linux-amd64
	cp ${BUILDDIR}/${PLUGIN_NAME}-linux-amd64${FILE_SUFFIX}.so ${PLUGINDIR}

setup-gotify: compose-up
	@echo "Setting up Gotify..."
	@for i in 1 2 3 4 5; do \
		echo "Attempt $$i of 5..."; \
		sleep 5; \
		NEW_TOKEN=$$(curl -s -f -X POST \
			-u admin:admin \
			-H "Content-Type: application/json" \
			-d '{"name":"test-client"}' \
			http://localhost:8888/client \
			| jq -r '.token'); \
		if [ -n "$$NEW_TOKEN" ]; then \
			sed -i '' 's/^TG_PLUGIN__GOTIFY_CLIENT_TOKEN=.*/TG_PLUGIN__GOTIFY_CLIENT_TOKEN='$$NEW_TOKEN'/' .env && \
			echo "TG_PLUGIN__GOTIFY_CLIENT_TOKEN updated in .env. Restarting gotify..." && \
			docker compose down && docker compose up -d && \
			exit 0; \
		fi; \
		echo "Attempt $$i failed. Retrying..."; \
	done; \
	echo "Failed to get token from Gotify after 5 attempts"; \
	exit 1

test-plugin-arm64: move-plugin-arm64 setup-gotify

test-plugin-amd64: move-plugin-amd64 setup-gotify

.PHONY: build check-env compose-up compose-down test
