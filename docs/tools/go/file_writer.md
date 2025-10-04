# File Writer Tool

## Description

The `file_writer` tool is a Go-based gRPC service that writes specified content to a file on the local filesystem.

> **Warning:** This tool will **overwrite** the file if it already exists. This action is irreversible and can lead to data loss. Use with caution.

## Security

For security, this tool has a built-in safeguard to prevent directory traversal attacks. It cleans the provided file path and denies any request that attempts to access parent directories using `..`. All file access is sandboxed within the project's working directory.

## Parameters

The tool accepts the following arguments in a JSON object:

| Parameter  | Type     | Required | Description                                                    |
|------------|----------|----------|----------------------------------------------------------------|
| `filepath` | `string` | **Yes**  | The path to the file to be written (e.g., `docs/new_notes.txt`). |
| `content`  | `string` | **Yes**  | The content to write to the file.                              |

## Response

*   **Successful Write:** If the file is written successfully, the tool returns a success message indicating the number of bytes written and the file path (e.g., `"Successfully wrote 50 bytes to docs/new_notes.txt"`).
*   **Error:** If the path is invalid, a write error occurs, or access is denied, the tool returns an error message.

## Usage Example

Here is an example of how to use the `file_writer` tool to create a new file and write content to it. This example uses `curl` to interact with the MCP server's HTTP/REST gateway.

**Request:**

```bash
curl -X POST http://localhost:8002/v1/tools:run \
-d '{
  "name": "file_writer",
  "arguments": {
    "filepath": "data/new_file.txt",
    "content": "This is a new line of text written by the tool."
  }
}'
```

**Successful Response:**

```json
{
  "result": {
    "stringValue": "Successfully wrote 49 bytes to data/new_file.txt"
  }
}
```

## Configuration

The tool's configuration is managed through a `config.json` file located in the tool's directory.

**Example `config.json`:**
```json
{
  "port": 50055,
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