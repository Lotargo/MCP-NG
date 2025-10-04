Integration Guide
This guide provides detailed instructions on how to integrate new and existing tools into the MCP (Master Control Program) server. It also covers how to connect the MCP server with client applications and leverage ReAct patterns for intelligent tool use.
Adding a New Tool to the MCP Server
Integrating a new tool into the MCP server involves creating a gRPC service for the tool and providing a configuration file so the main server can automatically launch and manage it. I have designed the system to be modular, so adding new tools is a straightforward process.
1. Define the Tool's gRPC Contract
First, update the gRPC contract in MCP-NG/proto/mcp.proto to include your new service definition. Maintaining a single .proto file serves as a single source of truth for the entire API.
Example: Adding a new ImageProcessor tool

```protobuf
// In MCP-NG/proto/mcp.proto

// ... other services

service ImageProcessor {
  rpc ProcessImage (ImageRequest) returns (ImageResponse);
}

message ImageRequest {
  bytes image_data = 1;
  string operation = 2; // e.g., "resize", "grayscale"
}

message ImageResponse {
  bytes processed_image_data = 1;
}
```

After updating the .proto file, regenerate the gRPC code for all languages by running the project's generation script from the root directory:

```bash
./generate_protos.sh
```

This script handles all dependencies and output paths automatically, ensuring that Go and Python stubs are correctly updated.

### 2. Implement the Tool's gRPC Server
Next, implement the gRPC server for your new tool. You can do this in Go or Python, following the structure of the existing tools.

*   **Go:** Create a new directory under `MCP-NG/tools/go/`.
*   **Python:** Create a new directory under `MCP-NG/tools/python/`.

The implementation should include the gRPC server logic and any business logic for the tool itself.

### 3. Implement the Health Check Service
Your tool **must** implement the standard gRPC Health Checking Protocol. This allows the main MCP server to monitor its status and route traffic only to healthy instances.

*   **Go:** Use the `google.golang.org/grpc/health` package.
*   **Python:** Use the `grpc_health.v1` package.

You should register the health service and set the initial status to `SERVING`.

### 4. Create the Configuration File
In the root of your tool's directory, create a `config.json` file. This file tells the main MCP server how to run and connect to your tool.

**Example `config.json`:**
```json
{
  "port": 50080,
  "command": ["go", "run", "."]
}
```
*   `port`: The port on which your tool's gRPC server will listen.
*   `command`: The command and arguments to execute your tool. The main server will run this command from your tool's directory.

## Deployment

### Using Docker (Recommended)

The easiest way to deploy and run the entire MCP-NG server is by using the provided Docker Compose setup. This handles the build process, starts the server, and ensures all components are correctly networked.

From the project root, simply run:
```bash
docker-compose up --build -d
```
The server will then be available at its default ports:
*   **HTTP/REST:** `http://localhost:8002`
*   **gRPC:** `localhost:8090`

This is the recommended method for both development and production environments as it guarantees a consistent and reproducible setup.

## Integrating with a Client Application
You can connect to the MCP-NG server using two primary methods: the simple HTTP/REST API or the high-performance native gRPC interface. For most use cases, especially for web clients or scripting, I recommend starting with the HTTP/REST API.

**Default Ports:**
*   **HTTP/REST:** `http://localhost:8002`
*   **gRPC:** `localhost:8090`

### The Easy Way: Using the HTTP/REST API
The gRPC-Gateway exposes a standard RESTful API that you can interact with using any HTTP client, such as `curl` or Python's `requests` library.

#### Example 1: Listing Available Tools with curl
Get a list of all healthy, available tools by making a `GET` request to the `/v1/tools` endpoint.

```bash
curl http://localhost:8002/v1/tools
```

**Example Response:**
```json
{
  "tools": [
    {
      "name": "calculator",
      "description": "A tool that evaluates mathematical expressions...",
      "parameters": {
        "type": "object",
        "properties": {
          "expression": { "type": "string", "description": "The mathematical expression..." }
        },
        "required": ["expression"]
      }
    },
    { "name": "web_search", "description": "Performs a web search...", "..." }
  ]
}
```

#### Example 2: Running a Tool with Python requests
To run a tool, send a `POST` request to the `/v1/tools:run` endpoint with the tool's name and arguments in the JSON body.

```python
import requests
import json

# The endpoint for running a tool
url = "http://localhost:8002/v1/tools:run"

# The data for the request
payload = {
  "name": "calculator",
  "arguments": {
    "expression": "(10 + 5) * 2"
  }
}

# Send the POST request
response = requests.post(url, json=payload)

# Print the result
print(response.json())
```
**Example Response:**
```json
{
  "result": {
    "stringValue": "30"
  }
}
```

### The Advanced Way: Using the Native gRPC Interface
For applications that require maximum performance and type safety, connecting directly to the gRPC server is the recommended approach. While this typically requires generating a client stub, you can easily test and debug the API using `grpcurl`.

#### Testing with `grpcurl`
`grpcurl` is a command-line tool that lets you interact with gRPC servers. It's like `curl`, but for gRPC.

##### 1. Installation
If you don't have it installed, you can get it with `go install`:
```bash
go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest
```
**Note:** Ensure your Go `bin` directory is in your system's `PATH`. If not, you can add `export PATH=$PATH:$(go env GOPATH)/bin` to your `.bashrc` or `.zshrc` file.

##### 2. List all available services
Because our server exposes the gRPC Reflection service, you can ask it what services it supports.
```bash
grpcurl -plaintext localhost:8090 list
```
**Expected Output:**
```
grpc.health.v1.Health
grpc.reflection.v1alpha.ServerReflection
mcp.MCP
```

##### 3. List methods for the MCP service
```bash
grpcurl -plaintext localhost:8090 list mcp.MCP
```
**Expected Output:**
```
mcp.MCP.GetHumanInput
mcp.MCP.ListTools
mcp.MCP.ProvideHumanInput
mcp.MCP.RunTool
```

##### 4. Call the `ListTools` method
```bash
grpcurl -plaintext localhost:8090 mcp.MCP.ListTools
```
This will return the same JSON list of tools as the `curl` example.

##### 5. Call `RunTool` with data
Use the `-d` flag to provide a JSON payload.
```bash
grpcurl -plaintext -d '{"name": "calculator", "arguments": {"expression": "(10 + 5) * 2"}}' localhost:8090 mcp.MCP.RunTool
```
**Example Response:**
```json
{
  "result": {
    "stringValue": "30"
  }
}
```
Using ReAct Patterns for Tool Selection
The ReAct (Reason and Act) pattern allows a large language model (LLM) to reason about which tool to use for a given task. This is a powerful way to build intelligent and autonomous agents.
Scenario: Dynamic Tool Selection
In this scenario, the LLM first requests a list of available tools from the MCP server and then decides which one to use based on the user's prompt. The MCP server will only return tools that are currently healthy.
User Prompt: "What's the weather in London?"
LLM Thought:
I need to find the weather. I should see what tools are available.
[LLM calls MCP server's ListTools method]
The web_search tool seems appropriate. I will use it to search for "weather in London".
[LLM calls MCP server's RunTool method with web_search]
By following this guide, you can extend the MCP server with new capabilities and integrate it into a variety of applications. The modular design and dual API make it a flexible and scalable platform for building intelligent agents.