FROM lukemathwalker/cargo-chef:latest-rust-1 AS chef
WORKDIR /app

FROM oven/bun:1 AS web-builder
WORKDIR /app/gateway/web
COPY gateway/web/package.json gateway/web/bun.lock ./
COPY gateway/web/index.html gateway/web/tsconfig.json gateway/web/vite.config.ts ./
COPY gateway/web/src src
COPY gateway/web/public public
COPY contracts /app/contracts
RUN bun install --frozen-lockfile
RUN bun run build

FROM golang:1.25-bookworm AS go-builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY cmd cmd
COPY internal internal
RUN CGO_ENABLED=0 go build -o /out/oneday ./cmd/oneday

FROM chef AS planner
COPY gateway/Cargo.toml gateway/Cargo.toml
COPY gateway/src gateway/src
WORKDIR /app/gateway
RUN cargo chef prepare --recipe-path recipe.json

FROM chef AS builder
COPY --from=planner /app/gateway/recipe.json recipe.json
RUN cargo chef cook --release --recipe-path recipe.json
COPY gateway/Cargo.toml Cargo.toml
COPY gateway/src src
RUN cargo build --release

FROM debian:trixie-slim AS runtime
RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates iproute2 \
    && rm -rf /var/lib/apt/lists/*
COPY --from=builder /app/target/release/oneday-gateway /usr/local/bin/oneday-gateway
COPY --from=go-builder /out/oneday /usr/local/bin/oneday
COPY --from=web-builder /app/gateway/web/dist /usr/local/share/oneday-gateway/web
ENV ONEDAY_BIN=/usr/local/bin/oneday
EXPOSE 8788
ENTRYPOINT ["oneday-gateway"]
