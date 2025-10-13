# Kademlia DHT Implementation in Go

A distributed hash table (DHT) implementation based on the Kademlia protocol, written in Go. This project provides both a CLI interface and a REST API for interacting with a Kademlia network, enabling distributed storage and retrieval of key-value pairs across multiple nodes.

## Table of Contents
- [Overview](#overview)
- [Features](#features)
- [Architecture](#architecture)
- [Quick Start](#quick-start)
- [Usage](#usage)
- [REST API](#rest-api)
- [Docker Deployment](#docker-deployment)
- [Development](#development)
- [Testing](#testing)

## Overview

Kademlia is a peer-to-peer distributed hash table that uses XOR distance metric for efficient routing. This implementation includes:
- **Distributed Storage**: Store and retrieve data across multiple nodes
- **Peer Discovery**: Automatic discovery and management of network peers
- **Fault Tolerance**: Data replication and automatic republishing
- **REST API**: HTTP interface for network operations
- **CLI Tools**: Command-line interface for network interaction

## Features

- ✅ Complete Kademlia protocol implementation
- ✅ UDP-based network communication with Protocol Buffers
- ✅ RESTful API for PUT, GET, DELETE operations
- ✅ CLI commands for network management
- ✅ Docker support with docker-compose orchestration
- ✅ Automatic data republishing and refreshing
- ✅ Configurable routing table parameters (k-bucket size, alpha)
- ✅ Comprehensive test coverage

## Architecture

### Project Structure
```
.
├── cmd/                    # Main application entry point
│   └── main.go            # Binary executable logic
├── internal/              # Private application code
│   └── cli/               # CLI command implementations
│       ├── start_node.go  # Start a Kademlia node
│       ├── put.go         # Store data in the network
│       ├── get.go         # Retrieve data from the network
│       ├── ping.go        # Ping nodes
│       ├── exit.go        # Exit from network
│       └── version.go     # Version information
├── pkg/                   # Public libraries
│   ├── kademlia/          # Core Kademlia implementation
│   │   ├── kademlia_node.go      # Node logic
│   │   ├── kademlia_restapi.go   # REST API endpoints
│   │   ├── common/               # Shared types (routing table, data objects)
│   │   └── rpc_handlers/         # RPC message handlers
│   ├── network/           # Network layer (UDP communication)
│   ├── utils/             # Utilities (BitArray, hashing, etc.)
│   └── proto_gen/         # Generated Protocol Buffer code
├── proto/                 # Protocol Buffer definitions
├── Makefile              # Build automation
├── Dockerfile            # Container image definition
├── docker-compose.yml    # Multi-node orchestration
└── compose_generator.py  # Generate docker-compose for N nodes
```

### Key Components

- **KademliaNode**: Core node implementation managing routing table, storage, and network operations
- **RoutingTable**: Manages k-buckets for efficient peer discovery
- **Network Layer**: UDP-based communication using Protocol Buffers for message serialization
- **RPC Handlers**: Handle protocol messages (PING, FIND_NODE, FIND_VALUE, STORE, etc.)
- **REST API**: HTTP interface for external applications

## Quick Start

### Prerequisites
- Go 1.23.5 or higher
- Docker (optional, for containerized deployment)
- Make (optional, for build automation)

### Build the Project

```bash
# Install dependencies
go mod tidy

# Build the binary
make build
```

This creates the binary at `./bin/helloworld`.

### Run a Single Node

```bash
# Start a Kademlia node on port 8001
./bin/helloworld start_node 8001
```

The node will:
- Listen for UDP messages on the specified port
- Start a REST API server on port + 1 (e.g., 8002 for HTTP)
- Save node information to `~/.kademlia/nodes/node_info.json`

### Interact with the Network

Once you have a node running, you can interact with it using CLI commands:

```bash
# Store a value
./bin/helloworld put "Hello, Kademlia!"
# Output: Value stored with key: <hex-encoded-key>

# Retrieve a value
./bin/helloworld get <key>
# Output: Value retrieved for key: Hello, Kademlia!

# Ping the node
./bin/helloworld ping
# Output: Ping successful! Response RPC ID: <id> from <node-id>

# Check version
./bin/helloworld version
# Output: <commit-hash>
#         <build-time>
```

## Usage

### CLI Commands

#### `start_node <port>`
Start a new Kademlia node listening on the specified port.

```bash
./bin/helloworld start_node 8001
```

The node will:
- Create a 160-bit random node ID
- Initialize the routing table with k=4 buckets
- Start the UDP listener and REST API server
- Bootstrap by connecting to known nodes in the network

#### `put <value>`
Store a value in the Kademlia network.

```bash
./bin/helloworld put "My data"
```

Returns the key (SHA-1 hash) where the value is stored. The CLI automatically:
- Reads local node information from `~/.kademlia/nodes/node_info.json`
- Creates a temporary node to send the PUT request
- Finds the K closest nodes to the key
- Stores the value on those nodes

#### `get <key>`
Retrieve a value from the network using its key.

```bash
./bin/helloworld get <hex-key>
```

Returns the value if found. The system performs a FIND_VALUE lookup, which:
- Iteratively queries nodes closer to the key
- Returns the value if found
- Returns closest nodes if not found

#### `ping`
Send a PING RPC to verify node connectivity.

```bash
./bin/helloworld ping
```

Verifies that the local node can communicate with the network.

#### `exit <key>`
Signal that this node is exiting the network.

```bash
./bin/helloworld exit <key>
```

Notifies the network that the node is leaving, allowing other nodes to update their routing tables.

#### `version`
Display build version and timestamp.

```bash
./bin/helloworld version
```

### Global Flags

- `-v, --verbose`: Enable verbose output for debugging

## REST API

The REST API server starts automatically when you run `start_node` on port `<node-port> + 1`.

### Endpoints

#### POST `/objects`
Store data in the network.

**Request:**
```json
{
  "data": "Hello, World!"
}
```

**Response (201 Created):**
```json
{
  "value": "Hello, World!"
}
```

**Headers:**
- `Location: /objects/<hex-key>`

#### GET `/objects?id=<hex-key>`
Retrieve data by key.

**Response (200 OK):**
```json
[
  {
    "data": "Hello, World!",
    "owner": "<node-id>",
    "timestamp": "2025-10-13T12:00:00Z"
  }
]
```

#### DELETE `/objects?id=<hex-key>`
Remove data from local storage (if this node is refreshing it).

**Response (204 No Content)**

#### POST `/exit`
Gracefully shut down the node.

**Response (200 OK)**

### Example cURL Commands

```bash
# Store data
curl -X POST http://localhost:8002/objects \
  -H "Content-Type: application/json" \
  -d '{"data":"Hello, Kademlia!"}'

# Get data
curl "http://localhost:8002/objects?id=<hex-key>"

# Delete data
curl -X DELETE "http://localhost:8002/objects?id=<hex-key>"

# Exit node
curl -X POST http://localhost:8002/exit
```

## Docker Deployment

### Single Container

Build and run a single node:

```bash
# Build image
make container
# or
docker build -t go-node:latest .

# Run a node
docker run -p 8001:8001/udp -p 8002:8002 go-node:latest start_node 8001
```

### Multi-Node Network

Use docker-compose to deploy a multi-node network:

```bash
# Deploy default configuration (50 nodes)
docker-compose up

# Scale to custom number of nodes
python compose_generator.py  # Edit NUM_NODES variable
docker-compose up
```

The compose configuration:
- Creates a custom network for inter-node communication
- Deploys nodes on sequential ports (8001, 8002, 8003, ...)
- Each node automatically discovers and connects to others

### Custom Network Size

Edit `compose_generator.py`:
```python
NUM_NODES = 10  # Set desired number of nodes
BASE_PORT = 8001
```

Then regenerate and deploy:
```bash
python compose_generator.py > docker-compose.yml
docker-compose up
```

## Development

### External Packages

This project uses:
- **[Cobra](https://github.com/spf13/cobra)**: CLI framework (used by Kubernetes, Docker, etc.)
- **[Logrus](https://github.com/sirupsen/logrus)**: Structured logging
- **[Protocol Buffers](https://github.com/protocolbuffers/protobuf)**: Message serialization
- **[Google UUID](https://github.com/google/uuid)**: UUID generation

### Building

```bash
# Build binary
make build

# Build Docker image
make container

# Install binary system-wide
make install
```

### Code Coverage

```bash
make coverage
```

This runs tests with coverage analysis across all packages.

## Testing

### Run All Tests

```bash
make test
```

This runs tests with the race detector enabled across:
- `pkg/kademlia`
- `pkg/kademlia/common`
- `pkg/kademlia/rpc_handlers`
- `pkg/network`
- `pkg/node`
- `pkg/utils`

### Run Tests for Specific Package

```bash
cd pkg/kademlia
go test -v --race
```

### Run Individual Test

```bash
cd pkg/kademlia
go test -v --race -run=TestKademliaNode
```

**Important:** Always use the `--race` flag to detect race conditions in concurrent code.

### Test Coverage

```bash
# Generate coverage report
make coverage

# View HTML coverage report
go tool cover -html=coverage.out
```

## Configuration

### Kademlia Parameters

Default configuration (in `start_node.go`):
- **k**: 4 (bucket size, number of nodes per k-bucket)
- **alpha**: 3 (parallelism parameter for lookups)
- **ID length**: 160 bits (compatible with SHA-1)
- **Republish interval**: 1 hour
- **Refresh interval**: 1 hour

These can be modified in the source code for experimentation.

### Node Storage

Node information is stored at:
```
~/.kademlia/nodes/node_info.json
```

This file contains:
```json
{
  "id": "<160-bit-id>",
  "ip": "<node-hostname>",
  "port": <udp-port>
}
```

## Continuous Integration

GitHub Actions automatically runs tests on push to `main`:
- `.github/workflows/go.yml`: Go tests
- `.github/workflows/docker_master.yaml`: Docker build
- `.github/workflows/releaser.yaml`: Release automation

**Releases:** Create a new release on GitHub to trigger automated binary builds via GoReleaser.

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run tests: `make test`
5. Submit a pull request

## License

This project is part of the D7024E Distributed Systems course at Luleå University of Technology.

## Acknowledgments

- Based on the [Kademlia paper](https://pdos.csail.mit.edu/~petar/papers/maymounkov-kademlia-lncs.pdf) by Maymounkov and Mazières
- Built following [Golang Standards Project Layout](https://github.com/golang-standards/project-layout)
