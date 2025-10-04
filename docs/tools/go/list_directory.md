# List Directory Tool

## Description

The `list_directory` tool is a Go-based gRPC service that lists the contents (files and subdirectories) of a specified directory on the local filesystem.

## Security

For security, this tool has a built-in safeguard to prevent directory traversal attacks. It cleans the provided file path and denies any request that attempts to access parent directories using `..`. All file access is sandboxed within the project's working directory.

## Parameters

The tool accepts a single optional argument in a JSON object:

| Parameter | Type     | Required | Default | Description                                                |
|-----------|----------|----------|---------|------------------------------------------------------------|
| `path`    | `string` | No       | `.`     | The path to the directory to list (e.g., `src/`). Defaults to the current working directory if not provided. |

## Response

*   **Successful List:** If the directory is found, the tool returns a JSON array of strings. Each string is a file or directory name.
    *   Subdirectories will have a trailing slash (e.g., `my_folder/`).
    *   Files will not have a trailing slash (e.g., `my_file.txt`).
*   **Error:** If the path does not exist or is not a directory, the tool returns an error message.

## Usage Example

Here is an example of how to use the `list_directory` tool to list the contents of the `docs` directory.

**Request:**

```bash
curl -X POST http://localhost:8002/v1/tools:run \
-d '{
  "name": "list_directory",
  "arguments": {
    "path": "docs/"
  }
}'
```

**Successful Response:**

```json
{
  "result": {
    "listValue": {
      "values": [
        { "stringValue": "tools/" },
        { "stringValue": "integration_guide.md" }
      ]
    }
  }
}
```

## Configuration

The tool's configuration is managed through a `config.json` file located in the tool's directory.

**Example `config.json`:**
```json
{
  "port": 50057,
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