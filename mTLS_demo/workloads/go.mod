# Module declaration
module github.com/yourusername/mtls-demo

# Go version requirement
go 1.21

# Direct dependencies
require (
    # HTTP server and routing
    github.com/gin-gonic/gin v1.9.1
    # Structured logging
    go.uber.org/zap v1.26.0
    # Prometheus metrics
    github.com/prometheus/client_golang v1.17.0
    # SPIFFE workload API
    github.com/spiffe/go-spiffe/v2 v2.1.6
    # TLS certificate handling
    github.com/spiffe/spire-api-sdk v1.8.5
    # HTTP client with circuit breaker
    github.com/sony/gobreaker v0.5.0
    # Retry mechanism
    github.com/cenkalti/backoff/v4 v4.2.1
    # Memory-mapped files
    github.com/edsrzf/mmap-go v1.1.0
)

# Indirect dependencies
require (
    # HTTP/2 support
    golang.org/x/net v0.19.0
    # TLS implementation
    golang.org/x/crypto v0.16.0
    # JSON processing
    github.com/json-iterator/go v1.1.12
    # Protocol buffers
    google.golang.org/protobuf v1.31.0
    # gRPC support
    google.golang.org/grpc v1.59.0
) 