# File: tests/test_code_interpreter.py
# Purpose: Test the 'code_interpreter' tool via the main gRPC server.

import grpc
import pytest
import subprocess
import time
import os
import sys
import select
from pathlib import Path

# Add proto directory to path to import generated files
proto_path = Path(__file__).parent.parent / 'proto'
sys.path.append(str(proto_path))

# Since the generated Python files are in `tools/python/code_interpreter`,
# and not in a standard package, we need to add that directory to the path as well.
tool_proto_path = Path(__file__).parent.parent / 'tools/python/code_interpreter'
sys.path.append(str(tool_proto_path))

import mcp_pb2
import mcp_pb2_grpc
from google.protobuf import struct_pb2

# --- Test Setup ---
SERVER_ADDRESS = "localhost:8002"
SERVER_CMD = ["go", "run", "main.go"]
SERVER_DIR = str(Path(__file__).parent.parent / 'server' / 'cmd' / 'server')

@pytest.fixture(scope="module")
def mcp_server():
    """
    Starts and stops the main MCP server for the test module.
    Waits for the server to be ready before yielding.
    """
    print("Starting MCP server...")
    server_process = subprocess.Popen(
        SERVER_CMD,
        cwd=SERVER_DIR,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
        preexec_fn=os.setsid
    )

    # Wait for the server to be ready by reading its stdout
    ready = False
    timeout = 120  # seconds
    start_time = time.time()
    log_buffer = ""

    while time.time() - start_time < timeout:
        # Use select for non-blocking reads on stdout/stderr
        readable, _, _ = select.select([server_process.stdout, server_process.stderr], [], [], 0.1)

        for stream in readable:
            line = stream.readline()
            if line:
                log_buffer += line
                print(f"Server log: {line.strip()}")
                if "gRPC server listening on :8090" in line:
                    print("Server is ready.")
                    ready = True
                    break

        if ready:
            break

        if server_process.poll() is not None:
            print("Server process terminated unexpectedly.")
            break

    if not ready:
        print("\n--- Server logs on failure ---")
        print(log_buffer)
        # Kill the process group if it's still running
        if server_process.poll() is None:
            os.killpg(os.getpgid(server_process.pid), 15)
        stdout, stderr = server_process.communicate()
        print(stdout)
        print(stderr)
        pytest.fail("MCP server failed to start within the timeout period.")

    yield
    
    print("Stopping MCP server...")
    os.killpg(os.getpgid(server_process.pid), 15)
    server_process.wait()
    print("Server stopped.")


@pytest.fixture(scope="module")
def grpc_client(mcp_server):
    """
    Provides a gRPC client for the MCP server.
    """
    channel = grpc.insecure_channel(SERVER_ADDRESS)
    client = mcp_pb2_grpc.MCPStub(channel)
    # Wait for channel to be ready
    grpc.channel_ready_future(channel).result(timeout=10)
    yield client
    channel.close()

def test_simple_print(grpc_client):
    """Test case for a simple print statement."""
    code = 'print("Hello from the interpreter!")'
    args = struct_pb2.Struct()
    args.fields["code"].string_value = code

    request = mcp_pb2.ToolRunRequest(name="code_interpreter", arguments=args)
    response = grpc_client.RunTool(request)

    assert not response.error
    assert "Hello from the interpreter!" in response.result.string_value

def test_calculation(grpc_client):
    """Test case for a simple calculation."""
    code = (
        "x = 10\n"
        "y = 25\n"
        "result = (x + y) * 2\n"
        "print(f'Result: {result}')"
    )
    args = struct_pb2.Struct()
    args.fields["code"].string_value = code

    request = mcp_pb2.ToolRunRequest(name="code_interpreter", arguments=args)
    response = grpc_client.RunTool(request)

    assert not response.error
    assert "Result: 70" in response.result.string_value

def test_code_error_handling(grpc_client):
    """Test case for handling an error in the executed code."""
    code = "result = 100 / 0"
    args = struct_pb2.Struct()
    args.fields["code"].string_value = code

    request = mcp_pb2.ToolRunRequest(name="code_interpreter", arguments=args)
    response = grpc_client.RunTool(request)

    assert response.error
    assert "ZeroDivisionError" in response.error