## Prerequisites

- Go 1.16 or later
- Protocol Buffers compiler (protoc)
- Go gRPC tools

## Installation

1. Install the required Go packages:
```bash
go mod init mutex
go get -u google.golang.org/grpc
go get -u google.golang.org/protobuf
```

2. Generate gRPC code:
```bash
protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    proto/mutex.proto
```

## Running the System

Open three terminal windows and run the following commands:

Terminal 1:
```bash
go run main.go -id 1 -port 50051 -peers "localhost:50051,localhost:50052,localhost:50053"
```

Terminal 2:
```bash
go run main.go -id 2 -port 50052 -peers "localhost:50051,localhost:50052,localhost:50053"
```

Terminal 3:
```bash
go run main.go -id 3 -port 50053 -peers "localhost:50051,localhost:50052,localhost:50053"
```

Each node will:
1. Start a gRPC server
2. Connect to other nodes
3. Periodically request access to the critical section
4. Log all messages and actions
