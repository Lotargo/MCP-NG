# Code Interpreter Tool

## Description

The `code_interpreter` tool is a Python-based gRPC service that executes a provided string of Python code and returns the output from `stdout`. It is a highly powerful but also dangerous tool that should be used with extreme caution.

> **Warning:** This tool is **EXTREMELY DANGEROUS**. It executes arbitrary code provided by an agent, which can lead to security vulnerabilities, data loss, or system instability. It should only be enabled in secure, sandboxed environments and with strict oversight. It is disabled by default in the main server configuration.

## Parameters

The tool accepts a single argument in a JSON object:

| Parameter | Type     | Required | Description                                  |
|-----------|----------|----------|----------------------------------------------|
| `code`    | `string` | **Yes**  | A string containing valid Python code to execute. |

## Response

*   **Successful Execution:** If the code runs without errors, the tool returns a JSON object containing a `result` field. The value of `result` is the captured standard output (`stdout`) from the executed code.
*   **Error:** If the code contains a syntax error or raises an exception during execution, the tool returns an error message containing the traceback.

## Usage Example

Here is an example of how to use the `code_interpreter` to perform a simple calculation and print the result.

**Request:**

```bash
curl -X POST http://localhost:8002/v1/tools:run \
-d '{
  "name": "code_interpreter",
  "arguments": {
    "code": "import math\nresult = math.sqrt(256)\nprint(f\'The square root of 256 is {result}\')"
  }
}'
```

**Successful Response:**

```json
{
  "result": {
    "stringValue": "The square root of 256 is 16.0\n"
  }
}
```

## Configuration

The tool's configuration is managed through a `config.json` file.

**Example `config.json`:**
```json
{
  "port": 50071,
  "command": ["python", "server.py"]
}
```

## R&D Module

The `code_interpreter` is considered an **R&D (Research & Development)** module. It is powerful but comes with significant security risks. By default, the main MCP server is configured **not** to launch this tool to maintain a stable and secure environment.

## Health and Logging

*   **Health Checks:** Implements the standard gRPC Health Checking Protocol.
*   **Logging:** Uses the standard Python `logging` module.