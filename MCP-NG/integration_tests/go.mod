module mcp-ng/integration_tests

go 1.24.3

replace mcp-ng/server => ../server

replace mcp-ng/human_input-tool => ../tools/go/human_input

require (
	github.com/gorilla/websocket v1.5.3
	google.golang.org/grpc v1.75.1
	google.golang.org/protobuf v1.36.10
	mcp-ng/human_input-tool v0.0.0-00010101000000-000000000000
	mcp-ng/server v0.0.0-00010101000000-000000000000
)

require (
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.27.2 // indirect
	golang.org/x/net v0.43.0 // indirect
	golang.org/x/sys v0.35.0 // indirect
	golang.org/x/text v0.28.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20250922171735-9219d122eba9 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250922171735-9219d122eba9 // indirect
)
