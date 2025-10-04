<h1>Integration Guide</h1>
<p>This guide provides detailed instructions on how to integrate new and existing tools into the MCP-NG server. It also covers how to connect the MCP server with client applications and leverage ReAct patterns for intelligent tool use.</p>
<h2>Adding a New Tool to the MCP Server</h2>
<p>Integrating a new tool involves creating a gRPC service and providing a simple configuration file. The main server is designed to automatically discover, launch, and manage any tool that follows this structure.</p>
<h3>1. Define the Tool's gRPC Contract</h3>
<p>All service definitions are centralized in <code>MCP-NG/proto/mcp.proto</code>. This acts as the single source of truth for the entire API.</p>
<p>To add a new tool, you will typically implement the existing <code>Tool</code> service interface rather than defining a whole new service. The core methods are already defined:</p>
<ul>
<li><code>GetDescription</code>: Returns the tool's name, description, and expected parameters.</li>
<li><code>Run</code>: Executes the tool with the provided arguments.</li>
</ul>
<p>If your tool requires a completely new interaction pattern, you can add a new service to the <code>.proto</code> file. After updating it, regenerate the gRPC code for all languages by running the project's generation script from the root directory:</p>
<pre><code>./generate_protos.sh</code></pre>
<p>This script handles all dependencies and output paths automatically, ensuring that Go and Python stubs are correctly updated.</p>
<h3>2. Implement the Tool's gRPC Server</h3>
<p>Next, implement the gRPC server for your new tool. You can do this in Go or Python, following the structure of the existing tools.</p>
<ul>
<li><strong>Go:</strong> Create a new directory under <code>MCP-NG/tools/go/</code>.</li>
<li><strong>Python:</strong> Create a new directory under <code>MCP-NG/tools/python/</code>.</li>
</ul>
<p>Your implementation must define the logic for the <code>GetDescription</code> and <code>Run</code> methods.</p>
<h3>3. Implement the Health Check Service</h3>
<p>Your tool <strong>must</strong> implement the standard gRPC Health Checking Protocol. This allows the main MCP server to monitor its status and route traffic only to healthy instances.</p>
<ul>
<li><strong>Go:</strong> Use the <code>google.golang.org/grpc/health</code> package.</li>
<li><strong>Python:</strong> Use the <code>grpc_health.v1</code> package.</li>
</ul>
<p>Register the health service and set the initial serving status to <code>SERVING</code>.</p>
<h3>4. Create the Configuration File</h3>
<p>In the root of your tool's directory, create a <code>config.json</code> file. This file tells the main MCP server how to run your tool. The configuration is now universal for both local and Docker environments.</p>
<p><strong>Example for a Go tool (`api_caller/config.json`):</strong></p>
<pre><code>{
"port": 50051,
"command": ["api_caller"]
}
</code></pre>
<p><strong>Example for a Python tool (`code_interpreter/config.json`):</strong></p>
<pre><code>{
"port": 50071,
"command": ["server.py"]
}
</code></pre>
<ul>
<li><code>port</code>: The port on which your tool's gRPC server will listen.</li>
<li><code>command</code>: A single-element array containing the name of the executable (for Go) or the entrypoint script (for Python). The main server will intelligently construct the full command path based on the environment.</li>
</ul>
<h2>Integrating with a Client Application</h2>
<p>You can connect to the MCP-NG server using two primary methods: the simple HTTP/REST API or the high-performance native gRPC interface. For most use cases, especially for web clients or scripting, starting with the HTTP/REST API is recommended.</p>
<p><strong>Default Ports:</strong></p>
<ul>
<li><strong>HTTP/REST:</strong> <code>http://localhost:8002</code></li>
<li><strong>gRPC:</strong> <code>localhost:8090</code></li>
</ul>
<h3>The Easy Way: Using the HTTP/REST API</h3>
<p>The gRPC-Gateway exposes a standard RESTful API that you can interact with using any HTTP client, such as <code>curl</code> or Python's <code>requests</code> library.</p>
<h4>Example 1: Listing Available Tools with curl</h4>
<p>Get a list of all healthy, available tools by making a <code>GET</code> request to the <code>/v1/tools</code> endpoint.</p>
<pre><code>curl http://localhost:8002/v1/tools</code></pre>
<p><strong>Example Response:</strong></p>
<pre><code>{
"tools": [
{
"name": "calculator",
"description": "A tool that evaluates mathematical expressions...",
"parameters": {
"type": "object",
"properties": { "expression": { "type": "string", "description": "..." } },
"required": ["expression"]
}
},
{ "name": "web_search", "description": "Performs a web search...", "..." }
]
}
</code></pre>
<h4>Example 2: Executing a Tool with Python requests</h4>
<p>To run a tool, send a <code>POST</code> request to the <code>/v1/tools/execute</code> endpoint with the tool's name and arguments in the JSON body.</p>
<pre><code>import requests
import json

url = "http://localhost:8002/v1/tools/execute"

payload = {
"tool_name": "calculator",
"arguments": {
"expression": "(10 + 5) * 2"
}
}

response = requests.post(url, json=payload)
print(response.json())
</code></pre>
<p><strong>Example Response:</strong></p>
<pre><code>{
"result": {
"result": { "numberValue": 30 }
}
}
</code></pre>
<h3>The Advanced Way: Using the Native gRPC Interface</h3>
<p>For applications that require maximum performance, connecting directly to the gRPC server is the best approach. You can easily test the API using <code>grpcurl</code>.</p>
<h4>Testing with `grpcurl`</h4>
<p><code>grpcurl</code> is a command-line tool that lets you interact with gRPC servers, similar to `curl` for HTTP.</p>
<h5>1. Installation</h5>
<p>The easiest way to install it is often through your system's package manager (e.g., <code>brew install grpcurl</code> on macOS, <code>choco install grpcurl</code> on Windows).</p>
<h5>2. List all available services</h5>
<p>Because our server exposes the gRPC Reflection service, you can ask it what services it supports.</p>
<pre><code>grpcurl -plaintext localhost:8090 list</code></pre>
<p><strong>Expected Output:</strong></p>
<pre><code>grpc.health.v1.Health
grpc.reflection.v1alpha.ServerReflection
mcp.MCP
</code></pre>
<h5>3. List methods for the MCP service</h5>
<pre><code>grpcurl -plaintext localhost:8090 list mcp.MCP</code></pre>
<h5>4. Call the `ListTools` method</h5>
<pre><code>grpcurl -plaintext localhost:8090 mcp.MCP.ListTools</code></pre>
<h5>5. Call `ExecuteTool` with data</h5>
<p>Use the <code>-d</code> flag to provide a JSON payload.</p>
<pre><code>grpcurl -plaintext -d '{"tool_name": "calculator", "arguments": {"expression": "(10 + 5) * 2"}}' localhost:8090 mcp.MCP.ExecuteTool</code></pre>
<p><strong>Example Response:</strong></p>
<pre><code>{
"result": {
"result": {
"numberValue": 30
}
}
}
</code></pre>
<h2>Using ReAct Patterns for Tool Selection</h2>
<p>The ReAct (Reason and Act) pattern allows a large language model (LLM) to reason about which tool to use for a given task, creating a loop of thought, action, and observation.</p>
<p><strong>User Prompt:</strong> "What is the result of 15 times 3, and who is the current president of France?"</p>
<p><strong>ReAct Loop Example:</strong></p>
<ol>
<li><strong>Thought:</strong> The user has two questions. I should solve the math problem first. I will look for a calculator tool.
<ul><li><strong>Action:</strong> Call <code>ListTools()</code> on the MCP Server.</li></ul>
</li>
<li><strong>Observation:</strong> The server returns a list including a "calculator" tool.</li>
<li><strong>Thought:</strong> Great, I will use the "calculator" tool with the expression "15 * 3".
<ul><li><strong>Action:</strong> Call <code>ExecuteTool(tool_name="calculator", arguments={"expression": "15 * 3"})</code>.</li></ul>
</li>
<li><strong>Observation:</strong> The tool returns a result of <code>{"result": 45}</code>.</li>
<li><strong>Thought:</strong> The first part is done. Now I need to find the president of France. The "web_search" tool seems appropriate.
<ul><li><strong>Action:</strong> Call <code>ExecuteTool(tool_name="web_search", arguments={"query": "current president of France"})</code>.</li></ul>
</li>
<li><strong>Observation:</strong> The tool returns a text snippet: "Emmanuel Macron is the current president of France."</li>
<li><strong>Thought:</strong> I have both answers. I can now form the final response.
<ul><li><strong>Final Answer to User:</strong> The result of 15 times 3 is 45. The current president of France is Emmanuel Macron.</li></ul>
</li>
</ol>
<p>By following this guide, you can extend the MCP server with new capabilities and integrate it into a variety of applications, building powerful and flexible intelligent agents.</p>