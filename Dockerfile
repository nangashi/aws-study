# ビルドイメージ
ARG GO_IMAGE=1.16-buster
# ビルド対象モジュール（デフォルト）
ARG MODULE=lambda1

# ===== ビルド =====
FROM golang:$GO_IMAGE as build-image
ARG MODULE
WORKDIR /go/src

# モジュール読込
COPY go.mod .
RUN go mod download

# ソースコピー
COPY src/common common
COPY src/$MODULE $MODULE
WORKDIR $MODULE

# ビルド
RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 \
    go build \
    -o /go/bin/main \
    -ldflags="-w -s"

# エントリーの定義
# ENTRYPOINT ["/go/bin/main"]

# ===== 実行イメージ作成 =====
# FROM public.ecr.aws/lambda/go:1 as run-image
FROM alpine:3

COPY --from=build-image /go/bin/main /app/main

ENTRYPOINT ["/app/main"]
