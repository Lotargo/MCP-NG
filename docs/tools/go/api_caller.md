# API Caller Tool

## Description

The `api_caller` tool is a versatile Go-based gRPC service that performs an HTTP request to a specified URL. It is designed for interacting with external services via REST APIs and supports GET, POST, PUT, and DELETE methods.

## Parameters

The tool accepts the following arguments in a JSON object:

| Parameter   | Type     | Required | Default | Description                                                                 |
|-------------|----------|----------|---------|-----------------------------------------------------------------------------|
| `url`       | `string` | **Yes**  |         | The full URL of the API endpoint to call.                                   |
| `method`    | `string` | No       | `GET`   | The HTTP method to use (e.g., `GET`, `POST`, `PUT`, `DELETE`).                |
| `headers`   | `object` | No       |         | An optional JSON object of key-value pairs to be sent as request headers.   |
| `json_body` | `object` | No       |         | An optional JSON object to be sent as the request body for `POST` or `PUT`. |

**Note:** When `json_body` is provided, the `Content-Type` header is automatically set to `application/json`.

## Response

The tool's response depends on the content of the API's reply:

*   **JSON Response:** If the API returns a valid JSON body, the tool will parse it and return a JSON object.
*   **Text Response:** If the API returns a non-JSON body (e.g., plain text, HTML), the tool will return it as a single string.
*   **Error:** If the HTTP request fails (e.g., network error, non-2xx status code) or has a timeout (15 seconds), the tool returns an error message.

## Usage Example

Here is an example of how to use the `api_caller` tool to post data to a hypothetical `/users` API endpoint. This example uses `curl` to interact with the MCP server's HTTP/REST gateway.

**Request:**

```bash
curl -X POST http://localhost:8002/v1/tools:run \
-d '{
  "name": "api_caller",
  "arguments": {
    "url": "https://api.example.com/v1/users",
    "method": "POST",
    "headers": {
      "Authorization": "Bearer your_api_key_here"
    },
    "json_body": {
      "name": "John Doe",
      "email": "john.doe@example.com"
    }
  }
}'
```

**Successful Response (JSON):**

If the API call is successful, the tool might return a JSON object like this:

```json
{
  "result": {
    "structValue": {
      "fields": {
        "id": { "stringValue": "user_12345" },
        "status": { "stringValue": "created" }
      }
    }
  }
}
```

## Configuration

The tool's configuration is managed through a `config.json` file located in the tool's directory.

**Example `config.json`:**
```json
{
  "port": 50051,
  "command": ["go", "run", "."]
}
```

*   `port`: The port on which the tool's gRPC server will listen.
*   `command`: The command and arguments to execute the tool.

## Health and Logging

*   **Health Checks:** This tool implements the standard gRPC Health Checking Protocol. The main MCP server will not route requests to it if it is unhealthy.
*   **Logging:** The tool uses a structured JSON logger (`slog`) to log all operations to standard output for observability.