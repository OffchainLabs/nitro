FROM node:16-alpine as contracts-builder

WORKDIR /app
COPY solgen/package.json solgen/
RUN cd solgen && yarn
COPY solgen solgen/
RUN cd solgen && yarn build

FROM alpine:20210804 as builder

WORKDIR /app

RUN apk add --update-cache go=1.17.4-r0 gcc=11.2.1_git20211128-r0 g++=11.2.1_git20211128-r0

COPY go.mod go.sum ./
COPY go-ethereum/go.mod go-ethereum/go.sum go-ethereum/
COPY fastcache/go.mod fastcache/go.sum fastcache/
RUN go mod download
COPY solgen solgen/
COPY go-ethereum go-ethereum/
COPY --from=contracts-builder app/solgen/artifacts solgen/artifacts/
RUN mkdir -p solgen/go/ && \
	go run solgen/gen.go
COPY . ./
RUN go build -v ./cmd/node && \
    go build -v ./cmd/deploy

FROM alpine:20210804
COPY --from=builder app/node .
COPY --from=builder app/deploy .
ENTRYPOINT [ "./node" ]
