# Keyword Extractor Tool

## Description

The `keyword_extractor` tool is a Python-based gRPC service that extracts the most relevant keywords and named entities (like people, organizations, and locations) from a given body of text.

It automatically detects the language of the text (English or Russian) and applies the best NLP model for the task:
*   **English:** Uses the `KeyBERT` library with a sentence-transformer model.
*   **Russian:** Uses the `Natasha` library, which is optimized for Russian morphology.

> **Note:** This tool loads machine learning models into memory upon startup, making it resource-intensive. It is considered an **R&D (Research & Development)** module and is disabled by default in the main server configuration to ensure a stable and fast-starting environment.

## Parameters

The tool accepts the following arguments in a JSON object:

| Parameter      | Type     | Required | Default | Description                                            |
|----------------|----------|----------|---------|--------------------------------------------------------|
| `text`         | `string` | **Yes**  |         | The body of text from which to extract keywords.       |
| `max_keywords` | `number` | No       | `10`    | The maximum number of keywords or phrases to return.   |

## Response

*   **Successful Extraction:** Returns a JSON array of strings, where each string is an extracted keyword or named entity.
*   **Error:** If the `text` parameter is missing or an error occurs during processing, the tool returns an error message.

## Usage Example

Here is an example of how to use the `keyword_extractor` tool to pull key terms from a technical description.

**Request:**

```bash
curl -X POST http://localhost:8002/v1/tools:run \
-d '{
  "name": "keyword_extractor",
  "arguments": {
    "text": "The Go programming language, also known as Golang, was designed at Google by Robert Griesemer, Rob Pike, and Ken Thompson. It is a statically typed, compiled language with built-in support for concurrency.",
    "max_keywords": 5
  }
}'
```

**Successful Response:**

```json
{
  "result": {
    "listValue": {
      "values": [
        { "stringValue": "Golang" },
        { "stringValue": "Go programming language" },
        { "stringValue": "Ken Thompson" },
        { "stringValue": "Rob Pike" },
        { "stringValue": "Google" }
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
  "port": 50073,
  "command": ["python", "server.py"]
}
```

## Health and Logging

*   **Health Checks:** Implements the standard gRPC Health Checking Protocol.
*   **Logging:** Uses the standard Python `logging` module.