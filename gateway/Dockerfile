FROM lukemathwalker/cargo-chef:latest-rust-1 AS chef
WORKDIR /app

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
EXPOSE 8788
ENTRYPOINT ["oneday-gateway"]
