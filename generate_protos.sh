#!/bin/bash
#
# This script ensures reliable and reproducible generation of gRPC and gRPC-Gateway code.
# It should be executed from the root directory of the project.

# Exit immediately if a command exits with a non-zero status.
set -e

# Add Go's bin directory to the PATH for this script's execution.
# This ensures that protoc can find the plugins (protoc-gen-go, etc.) installed by 'go install'.
export PATH=$PATH:$(go env GOPATH)/bin

echo "--- Step 1: Verifying and installing required Go tools ---"

# Check for the presence of the protobuf compiler.
if ! command -v protoc &> /dev/null
then
    echo "Error: 'protoc' compiler not found. Please install the Protocol Buffers compiler."
    exit 1
fi

# Install the necessary Go plugins for protoc.
echo "Installing/updating Go plugins for protoc..."
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest

echo "--- Step 2: Managing and locating .proto dependencies ---"

# The official googleapis repository contains all the .proto files we need.
# We will clone it into a local 'third_party' directory to make our build self-contained.
THIRD_PARTY_DIR="third_party"
GOOGLE_APIS_DIR="$THIRD_PARTY_DIR/googleapis"

if [ ! -d "$GOOGLE_APIS_DIR" ]; then
    echo "Google APIs protos not found. Cloning repository..."
    # We use a shallow clone (--depth 1) because we only need the latest files, not the entire history.
    git clone --depth 1 https://github.com/googleapis/googleapis "$GOOGLE_APIS_DIR"
else
    echo "Google APIs protos already exist locally."
fi

# We still need the path to grpc-gateway for its own internal protos (e.g., for OpenAPI options).
GW_PATH=$(go list -f '{{.Dir}}' -m github.com/grpc-ecosystem/grpc-gateway/v2)
if [ -z "$GW_PATH" ]; then
    echo "Error: Could not find module github.com/grpc-ecosystem/grpc-gateway/v2. Please run 'go mod download'."
    exit 1
fi

echo "Path to Google APIs protos resolved to: $GOOGLE_APIS_DIR"
echo "Path to grpc-gateway protos resolved to: $GW_PATH"


echo "--- Step 3: Generating Go code from .proto definitions ---"

# Define project-specific paths.
PROTO_DIR="MCP-NG/proto"
OUT_DIR="MCP-NG/server/pkg/mcp"

# Execute the protoc command with the correct include paths.
# -I "$GOOGLE_APIS_DIR" tells protoc to look for files like 'google/api/annotations.proto' inside our cloned repo.
protoc \
    -I "$PROTO_DIR" \
    -I "$GOOGLE_APIS_DIR" \
    -I "$GW_PATH/protoc-gen-openapiv2/options" \
    --go_out="$OUT_DIR" --go_opt=paths=source_relative \
    --go-grpc_out="$OUT_DIR" --go-grpc_opt=paths=source_relative \
    --grpc-gateway_out="$OUT_DIR" --grpc-gateway_opt=paths=source_relative \
    "$PROTO_DIR/mcp.proto"

echo "Code generation completed successfully."

echo "--- Step 4: Tidying Go module dependencies ---"

# Run 'go mod tidy' to ensure the go.mod and go.sum files are up-to-date.
(cd MCP-NG/server && go mod tidy)

echo "Server's Go module dependencies have been tidied."