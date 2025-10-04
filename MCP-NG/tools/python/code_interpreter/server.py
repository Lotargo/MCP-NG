import grpc
from concurrent import futures
import time
import sys
from pathlib import Path
import logging
import json

# Configure structured logging
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s - %(message)s')

# Add the parent directory to sys.path to allow imports if needed,
# This ensures that imports work regardless of how the script is run.
sys.path.append(str(Path(__file__).parent))

import mcp_pb2
import mcp_pb2_grpc
from google.protobuf.struct_pb2 import Value
from google.protobuf.json_format import ParseDict
from grpc_health.v1 import health
from grpc_health.v1 import health_pb2
from grpc_health.v1 import health_pb2_grpc

# Import the original tool logic
from code_interpreter import code_interpreter

class CodeInterpreterTool(mcp_pb2_grpc.ToolServicer):
    """
    gRPC servicer that wraps the original code_interpreter tool.
    """
    def GetDescription(self, request, context):
        """
        Returns the tool's description.
        """
        logging.info("Received request for tool description")
        description = mcp_pb2.ToolDescription(
            name="code_interpreter",
            description="Executes provided Python code and returns the result (stdout). EXTREMELY DANGEROUS! This tool executes arbitrary code.",
            parameters=mcp_pb2.ToolParameters(
                type="object",
                properties={
                    "code": mcp_pb2.ToolParameter(
                        type="string",
                        description="A string containing valid Python code."
                    )
                },
                required=["code"]
            )
        )
        return description

    def Run(self, request, context):
        """
        Executes the tool's logic.
        """
        logging.info(f"Received request to run code_interpreter with args: {request.arguments}")
        args = request.arguments

        code = args.fields.get("code").string_value
        if not code:
            logging.error("Invalid or missing 'code' argument")
            return mcp_pb2.ToolRunResponse(error="Invalid or missing 'code' argument")

        # Call the original Python function
        result_dict = code_interpreter(code)

        # Package the result into a gRPC response
        response = mcp_pb2.ToolRunResponse()
        if "error" in result_dict:
            logging.error(f"Error during code execution: {result_dict['error']}")
            response.error = str(result_dict["error"])
        elif "result" in result_dict:
            # Convert the python dict/value to a protobuf Value
            result_value = Value()
            ParseDict({"result": result_dict["result"]}, result_value)
            response.result.CopyFrom(result_value.struct_value.fields["result"])

        return response

def serve():
    """
    Starts the gRPC server.
    """
    config_path = Path(__file__).parent / 'config.json'
    try:
        with open(config_path, 'r') as f:
            config = json.load(f)
        port = config['port']
    except (FileNotFoundError, KeyError) as e:
        logging.critical(f"Failed to read or parse configuration: {e}")
        sys.exit(1)


    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    mcp_pb2_grpc.add_ToolServicer_to_server(CodeInterpreterTool(), server)

    # Add health check service
    health_servicer = health.HealthServicer()
    health_pb2_grpc.add_HealthServicer_to_server(health_servicer, server)
    health_servicer.set("mcp.Tool", health_pb2.HealthCheckResponse.SERVING)

    server.add_insecure_port(f'[::]:{port}')
    server.start()
    logging.info(f"Code Interpreter gRPC tool listening on :{port}")
    try:
        while True:
            time.sleep(86400) # One day
    except KeyboardInterrupt:
        logging.info("Shutting down server...")
        server.stop(0)

if __name__ == '__main__':
    serve()