# Web Search Tool

## Description

The `web_search` tool is a Go-based gRPC service that performs a web search using the **Tavily AI search engine**. It is designed to provide up-to-date information, facts, and news by returning a list of relevant search results.

## Parameters

The tool accepts the following arguments in a JSON object:

| Parameter     | Type     | Required | Default | Description                                            |
|---------------|----------|----------|---------|--------------------------------------------------------|
| `query`       | `string` | **Yes**  |         | The search query or question.                          |
| `max_results` | `number` | No       | `5`     | The maximum number of search results to return.        |

## Response

*   **Successful Search:** If the search is successful, the tool returns a JSON array of objects, where each object represents a search result from the Tavily API. Each result typically contains a `title`, `url`, `content` (snippet), and `score`.
*   **Error:** If the Tavily API key is not configured, the query is missing, or the API returns an error, the tool returns an error message.

## Usage Example

Here is an example of how to use the `web_search` tool to find information about the Go programming language.

**Request:**

```bash
curl -X POST http://localhost:8002/v1/tools:run \
-d '{
  "name": "web_search",
  "arguments": {
    "query": "What is the Go programming language?",
    "max_results": 2
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
              "title": { "stringValue": "The Go Programming Language" },
              "url": { "stringValue": "https://go.dev/" },
              "content": { "stringValue": "Go is an open source programming language that makes it easy to build simple, reliable, and efficient software." },
              "score": { "numberValue": 0.98 }
            }
          }
        },
        {
          "structValue": {
            "fields": {
              "title": { "stringValue": "Go (programming language) - Wikipedia" },
              "url": { "stringValue": "https://en.wikipedia.org/wiki/Go_(programming_language)" },
              "content": { "stringValue": "Go is a statically typed, compiled programming language designed at Google by Robert Griesemer, Rob Pike, and Ken Thompson..." },
              "score": { "numberValue": 0.97 }
            }
          }
        }
      ]
    }
  }
}
```

## Configuration

The tool requires a `config.json` file with the port and your Tavily API key.

**Example `config.json`:**
```json
{
  "port": 50060,
  "command": ["go", "run", "."],
  "tavily_api": {
    "api_key": "your-tavily-api-key"
  }
}
```

*   `port`: The port on which the tool's gRPC server will listen.
*   `command`: The command and arguments to execute the tool.
*   `tavily_api`: Your API key for the Tavily search service.

## Health and Logging

*   **Health Checks:** Implements the standard gRPC Health Checking Protocol.
*   **Logging:** Uses a structured JSON logger (`slog`) for observability.

## Testing

To test the tool, you can run the provided integration tests. Navigate to the tool's directory and run:
```bash
go test ./...
```