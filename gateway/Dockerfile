FROM lukemathwalker/cargo-chef:latest-rust-1@sha256:978393666bb054d0cb135a39b36cbdac82b5c0fb5553942e0ae7b0aa631f0e54 AS chef
WORKDIR /app

FROM oven/bun:1@sha256:e10577f0db68676a7024391c6e5cb4b879ebd17188ab750cf10024a6d700e5c4 AS web-builder
WORKDIR /app/gateway/web
COPY gateway/web/package.json gateway/web/bun.lock ./
COPY gateway/web/index.html gateway/web/tsconfig.json gateway/web/vite.config.ts ./
COPY gateway/web/src src
COPY gateway/web/public public
COPY contracts /app/contracts
RUN bun install --frozen-lockfile
RUN bun run build

FROM golang:1.25.12-bookworm@sha256:102fc956f6f6d98594cfe3cb6cd77b3d2699b6e914ab3ad6e819f3d8e83ebfac AS go-builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY cmd cmd
COPY internal internal
RUN CGO_ENABLED=0 go build -o /out/oneday ./cmd/oneday

FROM chef AS planner
COPY gateway/Cargo.toml gateway/Cargo.toml
COPY gateway/Cargo.lock gateway/Cargo.lock
COPY gateway/src gateway/src
WORKDIR /app/gateway
RUN cargo chef prepare --recipe-path recipe.json

FROM chef AS builder
WORKDIR /app/gateway
COPY --from=planner /app/gateway/recipe.json recipe.json
COPY gateway/Cargo.lock Cargo.lock
RUN cargo chef cook --release --recipe-path recipe.json
COPY gateway/Cargo.lock Cargo.lock
COPY gateway/Cargo.toml Cargo.toml
COPY gateway/build.rs build.rs
COPY gateway/src src
COPY contracts /app/contracts
RUN cargo build --release --locked

FROM debian:trixie-slim@sha256:020c0d20b9880058cbe785a9db107156c3c75c2ac944a6aa7ab59f2add76a7bd AS runtime
RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates iproute2 \
    && rm -rf /var/lib/apt/lists/*
COPY --from=builder /app/gateway/target/release/oneday-gateway /usr/local/bin/oneday-gateway
COPY --from=go-builder /out/oneday /usr/local/bin/oneday
COPY --from=web-builder /app/gateway/web/dist /usr/local/share/oneday-gateway/web
ENV ONEDAY_BIN=/usr/local/bin/oneday
EXPOSE 8788
ENTRYPOINT ["oneday-gateway"]
