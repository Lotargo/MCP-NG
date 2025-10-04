# Stage 1: Go Builder
# This stage builds the Go server, Go tools, and generates protobuf code.
FROM golang:1.24-alpine AS builder

# Install necessary tools: git for cloning protos, bash for the script,
# and protobuf-dev which includes the protoc compiler and the standard .proto files.
RUN apk add --no-cache git bash protobuf-dev

WORKDIR /app

# Copy Go workspace and module files
COPY go.work go.work.sum ./
COPY MCP-NG/ ./MCP-NG/

# Download all Go module dependencies for the workspace.
RUN go mod download

# Copy the proto generation script and run it
COPY generate_protos.sh .
RUN chmod +x generate_protos.sh
RUN ./generate_protos.sh

# Set the output directory for the Go binaries and add it to the PATH.
# 'go install' will automatically place the compiled executables here.
ENV GOBIN=/app/bin
ENV PATH=$GOBIN:$PATH

# Build and install all Go applications using 'go install'.
# This command is often more reliable than 'go build' for complex workspaces
# as it's specifically designed to compile and install executables.
# We also add flags to create smaller, static binaries.
ENV CGO_ENABLED=0
RUN go install -ldflags="-w -s" ./MCP-NG/server/cmd/server
RUN go install -ldflags="-w -s" ./MCP-NG/human_bridge
RUN go install -ldflags="-w -s" ./MCP-NG/tools/go/api_caller
RUN go install -ldflags="-w -s" ./MCP-NG/tools/go/calculator
RUN go install -ldflags="-w -s" ./MCP-NG/tools/go/db_querier
RUN go install -ldflags="-w -s" ./MCP-NG/tools/go/file_reader
RUN go install -ldflags="-w -s" ./MCP-NG/tools/go/file_writer
RUN go install -ldflags="-w -s" ./MCP-NG/tools/go/human_input
RUN go install -ldflags="-w -s" ./MCP-NG/tools/go/list_directory
RUN go install -ldflags="-w -s" ./MCP-NG/tools/go/log_notifier
RUN go install -ldflags="-w -s" ./MCP-NG/tools/go/ozon
RUN go install -ldflags="-w -s" ./MCP-NG/tools/go/web_search
RUN go install -ldflags="-w -s" ./MCP-NG/tools/go/wildberries

# Stage 2: Final Image
# This stage creates the final, lean image with the application and its dependencies.
FROM python:3.11-slim

WORKDIR /app

# Install system dependencies that Python packages might need
RUN apt-get update && apt-get install -y --no-install-recommends \
    git \
    procps \
    && rm -rf /var/lib/apt/lists/*

# Copy Python dependency requirements for Linux
COPY requirements_for_linux.txt .

# Install Python packages
RUN pip install --no-cache-dir -r requirements_for_linux.txt

# Copy the entire project structure for context
COPY MCP-NG/ ./MCP-NG/

# Copy the compiled Go binaries from the builder stage
COPY --from=builder /app/bin/ /app/bin/

# --- ИЗМЕНЕНИЕ: ДОБАВЛЕНО ЗДЕСЬ ---
# Устанавливаем PATH, чтобы главный сервер мог находить скомпилированные Go-инструменты
ENV PATH="/app/bin:${PATH}"

# Set the default command to start the main server
CMD ["/app/bin/server"]