# Text Generator Tool

## Description

The `text_generator` tool is a Python-based gRPC service that generates a short, pre-written block of text based on a selected topic. It is designed primarily for creating consistent test data for other parts of the system, not for dynamic or creative text generation.

> **Note:** This tool is a mock generator and does **not** use a Large Language Model (LLM). It is considered an **R&D (Research & Development)** module and is disabled by default in the main server configuration.

## Parameters

The tool accepts a single argument in a JSON object:

| Parameter | Type     | Required | Description                                                                 |
|-----------|----------|----------|-----------------------------------------------------------------------------|
| `topic`   | `string` | **Yes**  | The topic for which to generate text. Must be one of the following: `quantum_physics`, `python_programming`, `ancient_rome`. |

## Response

*   **Successful Generation:** If a valid topic is provided, the tool returns a string containing the pre-written text for that topic.
*   **Error:** If the `topic` parameter is missing or invalid, the tool returns an error message.

## Usage Example

Here is an example of how to use the `text_generator` tool to get the pre-written text about Python programming.

**Request:**

```bash
curl -X POST http://localhost:8002/v1/tools:run \
-d '{
  "name": "text_generator",
  "arguments": {
    "topic": "python_programming"
  }
}'
```

**Successful Response:**

```json
{
  "result": {
    "stringValue": "Python is a high-level, interpreted, general-purpose programming language. Its design philosophy emphasizes code readability with the use of significant indentation."
  }
}
```

## Configuration

The tool's configuration is managed through a `config.json` file.

**Example `config.json`:**
```json
{
  "port": 50074,
  "command": ["python", "server.py"]
}
```

## Health and Logging

*   **Health Checks:** Implements the standard gRPC Health Checking Protocol.
*   **Logging:** Uses the standard Python `logging` module.