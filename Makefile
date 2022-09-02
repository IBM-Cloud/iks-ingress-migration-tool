ifeq ($(OSTYPE),Darwin)
    platform := darwin
else
    platform := linux
endif

.PHONY: test
test:
	go test -race ./...

.PHONY: build
build:
	CGO_ENABLED=0 GOOS=$(platform) go build -a -tags netgo -ldflags '-w' -o ingress-migrator .
