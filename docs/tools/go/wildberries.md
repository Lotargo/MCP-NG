# Wildberries Tool

## Description

The `wildberries` tool is a versatile Go-based gRPC service that acts as a generic client for the **Wildberries API**. It allows an agent to make authenticated requests to any Wildberries API endpoint using various HTTP methods (`GET`, `POST`, `PUT`, `DELETE`, `PATCH`).

The tool automatically attaches the `Authorization` header with the API key from its configuration file to every request.

## Parameters

The tool accepts the following arguments in a JSON object:

| Parameter      | Type     | Required | Description                                                                     |
|----------------|----------|----------|---------------------------------------------------------------------------------|
| `method`       | `string` | **Yes**  | The HTTP method to use (e.g., `GET`, `POST`).                                   |
| `endpoint`     | `string` | **Yes**  | The API endpoint path, which must start with a `/` (e.g., `/api/v3/orders`).      |
| `query_params` | `object` | No       | An optional JSON object of key-value pairs to be sent as URL query parameters for `GET` requests. |
| `json_body`    | `object` | No       | An optional JSON object or array for the request body (for `POST`, `PUT`, `PATCH` requests). |

## Response

*   **Successful Request:** If the API call is successful, the tool returns the response from the Wildberries API.
    *   If the response is valid JSON, it is returned as a JSON object or array.
    *   If the response is not JSON or is empty, a success message is returned.
*   **Error:** If the API returns an error, the credentials are not configured, or another error occurs, the tool returns an error message.

## Usage Example

Here is an example of how to use the `wildberries` tool to get a list of new orders. This corresponds to a `GET` request to the `/api/v3/orders` endpoint in the Wildberries API documentation.

**Request:**

```bash
curl -X POST http://localhost:8002/v1/tools:run \
-d '{
  "name": "wildberries",
  "arguments": {
    "method": "GET",
    "endpoint": "/api/v3/orders",
    "query_params": {
      "limit": 10,
      "next": 0,
      "dateFrom": 1672531200
    }
  }
}'
```

**Successful Response (from Wildberries API):**

```json
{
  "result": {
    "structValue": {
      "fields": {
        "next": { "numberValue": 10 },
        "orders": {
          "listValue": {
            "values": [
              { "structValue": { "fields": { "id": { "numberValue": 12345 }, "article": { "stringValue": "PRODUCT-A" } } } },
              { "structValue": { "fields": { "id": { "numberValue": 12346 }, "article": { "stringValue": "PRODUCT-B" } } } }
            ]
          }
        }
      }
    }
  }
}
```

## Configuration

The tool requires a `config.json` file with its port and your Wildberries API key.

**Example `config.json`:**
```json
{
  "port": 50061,
  "command": ["go", "run", "."],
  "wildberries_api": {
    "api_key": "your-standard-wildberries-api-key"
  }
}
```

## Health and Logging

*   **Health Checks:** Implements the standard gRPC Health Checking Protocol.
*   **Logging:** Uses a structured JSON logger (`slog`) for observability.

## Testing

To test the tool, you can run the provided integration tests. Navigate to the tool's directory and run:
```bash
go test ./...
```