# DB Querier Tool

## Description

The `db_querier` tool is a Go-based gRPC service that executes a raw SQL query against a specified **SQLite** database file. It is designed for reading data but can execute any valid SQL statement.

> **Warning:** This tool can execute any SQL query provided to it, including destructive ones like `UPDATE`, `DELETE`, or `DROP TABLE`. Use with extreme caution, as it can lead to permanent data loss.

## Parameters

The tool accepts the following arguments in a JSON object:

| Parameter | Type     | Required | Description                                                  |
|-----------|----------|----------|--------------------------------------------------------------|
| `db_path` | `string` | **Yes**  | The local file path to the SQLite database (e.g., `./data/main.db`). |
| `query`   | `string` | **Yes**  | The SQL query to execute.                                    |

For security, the tool verifies that the database file exists before attempting to connect.

## Response

*   **Successful Query:** If the query executes successfully, the tool returns a JSON array of objects. Each object represents a row, with keys corresponding to the column names. An empty array `[]` is returned if the query yields no results.
*   **Error:** If the database file is not found, the SQL query is invalid, or another error occurs, the tool returns an error message.

## Usage Example

Here is an example of how to use the `db_querier` tool to select all users from a `users` table in a SQLite database file named `project.db`.

**Request:**

```bash
curl -X POST http://localhost:8002/v1/tools:run \
-d '{
  "name": "db_querier",
  "arguments": {
    "db_path": "data/project.db",
    "query": "SELECT id, name, email FROM users WHERE status = 'active';"
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
              "id": { "numberValue": 1 },
              "name": { "stringValue": "Alice" },
              "email": { "stringValue": "alice@example.com" }
            }
          }
        },
        {
          "structValue": {
            "fields": {
              "id": { "numberValue": 2 },
              "name": { "stringValue": "Bob" },
              "email": { "stringValue": "bob@example.com" }
            }
          }
        }
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
  "port": 50053,
  "command": ["go", "run", "."]
}
```

*   `port`: The port on which the tool's gRPC server will listen.
*   `command`: The command and arguments to execute the tool.

## Health and Logging

*   **Health Checks:** This tool implements the standard gRPC Health Checking Protocol.
*   **Logging:** The tool uses a structured JSON logger (`slog`) for observability.