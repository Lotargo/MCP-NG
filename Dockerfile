# Stage 1: Go Builder
# This stage builds the Go server, Go tools, and generates protobuf code.
FROM golang:1.24-alpine AS builder

# Install necessary tools: git for cloning protos, bash for the script,
# and protobuf-dev which includes the protoc compiler.
RUN apk add --no-cache git bash protobuf-dev

WORKDIR /app

# Copy Go workspace and module files
COPY go.work go.work.sum ./
COPY MCP-NG/ ./MCP-NG/

# Sync Go workspace modules for reliability.
RUN go work sync

# Copy the proto generation script and run it.
COPY generate_protos.sh .
RUN chmod +x generate_protos.sh && ./generate_protos.sh

# Create the output directory for the Go binaries.
RUN mkdir /app/bin

# Compile all Go applications using 'go build' with explicit output paths.
# This creates smaller, static binaries.
ENV CGO_ENABLED=0
RUN go build -ldflags="-w -s" -o /app/bin/server ./MCP-NG/server/cmd/server
RUN go build -ldflags="-w -s" -o /app/bin/human_bridge ./MCP-NG/human_bridge
RUN go build -ldflags="-w -s" -o /app/bin/api_caller ./MCP-NG/tools/go/api_caller
RUN go build -ldflags="-w -s" -o /app/bin/calculator ./MCP-NG/tools/go/calculator
RUN go build -ldflags="-w -s" -o /app/bin/db_querier ./MCP-NG/tools/go/db_querier
RUN go build -ldflags="-w -s" -o /app/bin/file_reader ./MCP-NG/tools/go/file_reader
RUN go build -ldflags="-w -s" -o /app/bin/file_writer ./MCP-NG/tools/go/file_writer
RUN go build -ldflags="-w -s" -o /app/bin/human_input ./MCP-NG/tools/go/human_input
RUN go build -ldflags="-w -s" -o /app/bin/list_directory ./MCP-NG/tools/go/list_directory
RUN go build -ldflags="-w -s" -o /app/bin/log_notifier ./MCP-NG/tools/go/log_notifier
RUN go build -ldflags="-w -s" -o /app/bin/ozon ./MCP-NG/tools/go/ozon
RUN go build -ldflags="-w -s" -o /app/bin/web_search ./MCP-NG/tools/go/web_search
RUN go build -ldflags="-w -s" -o /app/bin/wildberries ./MCP-NG/tools/go/wildberries

# Stage 2: Final Image
# This stage creates the final, lean image with the application and its dependencies.
FROM python:3.11-slim

WORKDIR /app

# Install system dependencies that Python packages might need.
RUN apt-get update && apt-get install -y --no-install-recommends \
    git \
    procps \
    && rm -rf /var/lib/apt/lists/*

# Copy Python dependency requirements.
COPY requirements_for_linux.txt .

# Create a virtual environment and install Python packages into it.
RUN python3 -m venv .venv
RUN . .venv/bin/activate && pip install --no-cache-dir -r requirements_for_linux.txt

# Copy the Go workspace files from the builder stage for project root discovery.
COPY --from=builder /app/go.work /app/go.work.sum ./

# Copy the entire project structure for context.
COPY MCP-NG/ ./MCP-NG/

# Copy the compiled Go binaries from the builder stage.
COPY --from=builder /app/bin/ /app/bin/

# Set the PATH to include the compiled Go binaries.
ENV PATH="/app/bin:${PATH}"

# Set the default command to start the main server.
CMD ["/app/bin/server"]