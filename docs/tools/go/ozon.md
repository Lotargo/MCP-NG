# Ozon Tool

## Description

The `ozon` tool is a Go-based gRPC service that acts as a generic client for the **Ozon Seller API**. It allows an agent to make authenticated `POST` requests to any Ozon API endpoint by providing the endpoint path and a JSON payload.

The tool automatically attaches the `Client-Id` and `Api-Key` from its configuration file to every request.

## Parameters

The tool accepts the following arguments in a JSON object:

| Parameter  | Type     | Required | Description                                                                |
|------------|----------|----------|----------------------------------------------------------------------------|
| `endpoint` | `string` | **Yes**  | The API endpoint path to call (e.g., `/v2/product/info`).                   |
| `payload`  | `object` | **Yes**  | A JSON object representing the request body to be sent to the Ozon API. |

## Response

*   **Successful Request:** If the API call is successful (HTTP 200 OK), the tool returns the JSON response from the Ozon API directly.
*   **Error:** If the API returns a non-200 status code, the credentials are not configured, or another error occurs, the tool returns an error message.

## Usage Example

Here is an example of how to use the `ozon` tool to get information about a specific product by its `product_id`. This corresponds to the `/v2/product/info` endpoint in the Ozon Seller API documentation.

**Request:**

```bash
curl -X POST http://localhost:8002/v1/tools:run \
-d '{
  "name": "ozon",
  "arguments": {
    "endpoint": "/v2/product/info",
    "payload": {
      "product_id": 123456789
    }
  }
}'
```

**Successful Response (from Ozon API):**

```json
{
  "result": {
    "structValue": {
      "fields": {
        "result": {
          "structValue": {
            "fields": {
              "id": { "numberValue": 123456789 },
              "name": { "stringValue": "Example Product Name" },
              "offer_id": { "stringValue": "PROD-001" },
              "barcode": { "stringValue": "A1B2C3D4E5" },
              "price": { "stringValue": "1999.00" }
            }
          }
        }
      }
    }
  }
}
```

## Configuration

The tool requires a `config.json` file with the port and Ozon Seller API credentials.

**Example `config.json`:**
```json
{
  "port": 50059,
  "command": ["go", "run", "."],
  "ozon_api": {
    "client_id": "your-client-id-from-ozon",
    "api_key": "your-api-key-from-ozon"
  }
}
```

*   `port`: The port on which the tool's gRPC server will listen.
*   `command`: The command and arguments to execute the tool.
*   `ozon_api`: Your Ozon Seller API credentials.

## Health and Logging

*   **Health Checks:** Implements the standard gRPC Health Checking Protocol.
*   **Logging:** Uses a structured JSON logger (`slog`) for observability.

## Testing

To test the tool, you can run the provided integration tests. Navigate to the tool's directory and run:
```bash
go test ./...
```