# Calculator Tool

## Description

The `calculator` tool is a Go-based gRPC service that evaluates mathematical expressions from a string. It leverages the `govaluate` library to support basic arithmetic operations (`+`, `-`, `*`, `/`), parentheses for order of operations, and many other common mathematical functions.

## Parameters

The tool accepts a single argument in a JSON object:

| Parameter    | Type     | Required | Description                                                    |
|--------------|----------|----------|----------------------------------------------------------------|
| `expression` | `string` | **Yes**  | The mathematical expression to evaluate (e.g., `"(2 + 2) * 4"`). |

## Response

*   **Successful Calculation:** If the expression is valid, the tool returns the resulting number.
*   **Error:** If the expression is malformed or contains an error during evaluation, the tool returns an error message.

## Usage Example

Here is an example of how to use the `calculator` tool to evaluate a complex expression. This example uses `curl` to interact with the MCP server's HTTP/REST gateway.

**Request:**

```bash
curl -X POST http://localhost:8002/v1/tools:run \
-d '{
  "name": "calculator",
  "arguments": {
    "expression": "(12.5 + 7.5) * (10 / 2)"
  }
}'
```

**Successful Response:**

```json
{
  "result": {
    "numberValue": 100
  }
}
```

## Configuration

The tool's configuration is managed through a `config.json` file located in the tool's directory.

**Example `config.json`:**
```json
{
  "port": 50052,
  "command": ["go", "run", "."]
}
```

*   `port`: The port on which the tool's gRPC server will listen.
*   `command`: The command and arguments to execute the tool.

## Health and Logging

*   **Health Checks:** This tool implements the standard gRPC Health Checking Protocol. The main MCP server will not route requests to it if it is unhealthy.
*   **Logging:** The tool uses a structured JSON logger (`slog`) to log all operations to standard output.

## Testing

To test the Calculator tool, you can run the provided integration tests. These tests cover various expressions and edge cases. Navigate to the tool's directory and run:
```bash
go test ./...
```