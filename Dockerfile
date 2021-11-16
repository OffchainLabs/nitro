FROM node:16-alpine as contracts-builder

WORKDIR /app
COPY solgen/package.json solgen/
RUN cd solgen && yarn
COPY solgen solgen/
RUN cd solgen && yarn build

FROM alpine:20210804 as builder

WORKDIR /app

RUN apk add --update-cache go=1.17.3-r0 gcc=10.3.1_git20211027-r0 g++=10.3.1_git20211027-r0

COPY go.mod go.sum ./
COPY go-ethereum/go.mod go-ethereum/go.sum go-ethereum/
COPY fastcache/go.mod fastcache/go.sum fastcache/
RUN go mod download
COPY . ./
COPY --from=contracts-builder app/solgen/artifacts solgen/artifacts/
RUN mkdir -p solgen/go/ && \
	go run solgen/gen.go && \
    go build ./cmd/node

FROM alpine:20210804
COPY --from=builder app/node .
ENTRYPOINT [ "./node" ]
