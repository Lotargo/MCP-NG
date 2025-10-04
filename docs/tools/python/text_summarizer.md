# Text Summarizer Tool

## Description

The `text_summarizer` tool is a Python-based gRPC service that creates a concise, abstractive summary of a longer text. It uses a pre-trained T5 (Text-to-Text Transfer Transformer) model to generate a summary that captures the main points of the source text.

> **Note:** This tool loads a large machine learning model into memory, making it resource-intensive. The model is lazy-loaded on the first call to the tool. It is considered an **R&D (Research & Development)** module and is disabled by default in the main server configuration to ensure a stable and fast-starting environment.

## Parameters

The tool accepts the following arguments in a JSON object:

| Parameter    | Type     | Required | Default | Description                                            |
|--------------|----------|----------|---------|--------------------------------------------------------|
| `text`       | `string` | **Yes**  |         | The body of text to be summarized.                     |
| `max_length` | `number` | No       | `150`   | The maximum length (in tokens) of the generated summary. |
| `min_length` | `number` | No       | `20`    | The minimum length (in tokens) of the generated summary. |

## Response

*   **Successful Summarization:** Returns a string containing the generated summary.
*   **Error:** If the `text` parameter is missing or an error occurs during model inference, the tool returns an error message.

## Usage Example

Here is an example of how to use the `text_summarizer` to summarize a paragraph.

**Request:**

```bash
curl -X POST http://localhost:8002/v1/tools:run \
-d '{
  "name": "text_summarizer",
  "arguments": {
    "text": "The James Webb Space Telescope (JWST) is a space telescope designed primarily to conduct infrared astronomy. As the largest optical telescope in space, its high resolution and sensitivity allow it to view objects too old, distant, or faint for the Hubble Space Telescope. This has enabled a broad range of investigations across many fields of astronomy and cosmology, such as observation of the first stars and the formation of the first galaxies.",
    "max_length": 50
  }
}'
```

**Successful Response:**

```json
{
  "result": {
    "stringValue": "The James Webb Space Telescope (JWST) is the largest optical telescope in space. It is designed to conduct infrared astronomy and view objects too old, distant, or faint for the Hubble Space Telescope."
  }
}
```

## Configuration

The tool's configuration is managed through a `config.json` file.

**Example `config.json`:**
```json
{
  "port": 50075,
  "command": ["python", "server.py"]
}
```

## Health and Logging

*   **Health Checks:** Implements the standard gRPC Health Checking Protocol.
*   **Logging:** Uses the standard Python `logging` module.