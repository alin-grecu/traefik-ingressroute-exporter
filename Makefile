CMD := traefik-ingressroute-exporter
GOOS ?= darwin
GOARCH ?= arm64

build:
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o out/$(GOOS)/$(GOARCH)/$(CMD)
