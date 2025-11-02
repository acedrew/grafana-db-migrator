FROM golang:1.18 AS builder
WORKDIR /go/src/github.com/percona/grafana-db-migrator/
COPY . .
# Build for the target platform architecture
ARG TARGETOS=linux
ARG TARGETARCH
RUN make OS=${TARGETOS} ARCH=${TARGETARCH}

FROM golang:1.18
WORKDIR /root/
COPY --from=builder /go/src/github.com/percona/grafana-db-migrator/dist/grafana-db-migrator ./grafana-migrate
RUN apt-get update && apt-get install -y \
    sqlite3 \
 && rm -rf /var/lib/apt/lists/*
ENTRYPOINT ["./grafana-migrate"]
