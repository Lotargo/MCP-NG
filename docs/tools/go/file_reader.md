# File Reader Tool

## Description

The `file_reader` tool is a Go-based gRPC service that reads the entire content of a specified file from the local filesystem and returns it as a single string.

## Security

For security, this tool has a built-in safeguard to prevent directory traversal attacks. It cleans the provided file path and denies any request that attempts to access parent directories using `..`. All file access is sandboxed within the project's working directory.

## Parameters

The tool accepts a single argument in a JSON object:

| Parameter  | Type     | Required | Description                                                    |
|------------|----------|----------|----------------------------------------------------------------|
| `filepath` | `string` | **Yes**  | The path to the file to be read (e.g., `src/main.go`). |

## Response

*   **Successful Read:** If the file is found and read successfully, the tool returns the entire content of the file as a single string.
*   **Error:** If the file does not exist, the path is invalid, or a read error occurs, the tool returns an error message.

## Usage Example

Here is an example of how to use the `file_reader` tool to read the contents of a project's `README.md`. This example uses `curl` to interact with the MCP server's HTTP/REST gateway.

**Request:**

```bash
curl -X POST http://localhost:8002/v1/tools:run \
-d '{
  "name": "file_reader",
  "arguments": {
    "filepath": "README.md"
  }
}'
```

**Successful Response:**

```json
{
  "result": {
    "stringValue": "# MCP-NG: A Go-Powered Server for the Model Context Protocol\n\nMCP-NG is a high-performance, modular server..."
  }
}
```

## Configuration

The tool's configuration is managed through a `config.json` file located in the tool's directory.

**Example `config.json`:**
```json
{
  "port": 50054,
  "command": ["go", "run", "."]
}
```

*   `port`: The port on which the tool's gRPC server will listen.
*   `command`: The command and arguments to execute the tool.

## Health and Logging

*   **Health Checks:** This tool implements the standard gRPC Health Checking Protocol.
*   **Logging:** The tool uses a structured JSON logger (`slog`) for observability.

## Testing

To test the tool, you can run the provided integration tests. Navigate to the tool's directory and run:
```bash
go test ./...
```