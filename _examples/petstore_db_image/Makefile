# Makefile
.PHONY: docker-build

# ----- settings you can tweak -----
IMG ?= petstore
TAG ?= latest
CTX ?= .
PLATFORMS ?= linux/amd64,linux/arm64
DOCKER_BUILDKIT ?= 1
# ----------------------------------

define DOCKERFILE
FROM alpine AS base
RUN apk add --no-cache curl wget && apk --no-cache add tzdata

FROM golang:1.24 AS go-builder
WORKDIR /go/app
COPY . /go/app
RUN GO111MODULE=on CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /go/app/main /go/app/cmd/app/main.go

FROM base
COPY --from=go-builder /go/app/main /main
EXPOSE 8080
CMD ["/main"]
endef
export DOCKERFILE

docker-build:
	@printf '%s\n' "$$DOCKERFILE" | DOCKER_BUILDKIT=$(DOCKER_BUILDKIT) \
		docker build --progress=plain \
		-t $(IMG):$(TAG) -f - $(CTX)

