# Hybrid Search Tool

## Description

The `hybrid_search` tool is a Python-based gRPC service that performs a sophisticated two-stage search on a local document collection. It is designed to find the most relevant documents by combining modern semantic search with traditional keyword filtering.

The tool uses a local sentence-transformer model for semantic search and a SQLite database for keyword-based filtering.

> **Note:** This tool loads a machine learning model into memory upon startup, making it resource-intensive. It is considered an **R&D (Research & Development)** module and is disabled by default in the main server configuration to ensure a stable and fast-starting environment.

## Hybrid Search Workflow

1.  **Semantic Search:** The tool first takes a `semantic_query` and uses a vector similarity search to find documents that are conceptually related to the query text.
2.  **Keyword Filtering:** The results from the semantic search are then filtered down using a SQL `WHERE` clause based on the optional `filters` provided.

## Parameters

The tool accepts the following arguments in a JSON object:

| Parameter        | Type     | Required | Description                                                                 |
|------------------|----------|----------|-----------------------------------------------------------------------------|
| `semantic_query` | `string` | **Yes**  | The natural language query for the semantic search (e.g., "information about Go programming"). |
| `filters`        | `object` | No       | An optional JSON object of key-value pairs to be used as `AND` conditions in a SQL `WHERE` clause (e.g., `{"author": "Rob Pike"}`). |

## Response

*   **Successful Search:** Returns a JSON array of objects, where each object represents a document matching both the semantic query and the filters. Each document includes its `id`, `author`, `title`, and `text`.
*   **Error:** If the `semantic_query` is missing or another error occurs, the tool returns an error message.

## Usage Example

Here is an example of searching for documents about "gRPC in Go" written by the author "Jane Doe".

**Request:**

```bash
curl -X POST http://localhost:8002/v1/tools:run \
-d '{
  "name": "hybrid_search",
  "arguments": {
    "semantic_query": "How to use gRPC in the Go language",
    "filters": {
      "author": "Jane Doe"
    }
  }
}'
```

**Successful Response:**

```json
{
  "result": {
    "listValue": {
      "values": [
        {
          "structValue": {
            "fields": {
              "id": { "numberValue": 101 },
              "author": { "stringValue": "Jane Doe" },
              "title": { "stringValue": "Advanced gRPC in Go" },
              "text": { "stringValue": "This document covers advanced patterns for using gRPC in a Go microservices architecture..." }
            }
          }
        }
      ]
    }
  }
}
```

## Configuration

The tool's configuration is managed through a `config.json` file.

**Example `config.json`:**
```json
{
  "port": 50072,
  "command": ["python", "server.py"]
}
```

## Health and Logging

*   **Health Checks:** Implements the standard gRPC Health Checking Protocol.
*   **Logging:** Uses the standard Python `logging` module.