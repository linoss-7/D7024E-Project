import yaml

NUM_NODES = 50
BASE_PORT = 8001
IMAGE_NAME = "go-node:latest"

services = {}

services["node_builder"] = {
    "build": ".",
    "image": IMAGE_NAME,
    "command": ["start_node", str(BASE_PORT)],
    "ports": [f"{BASE_PORT}:{BASE_PORT}/udp"],
    "deploy": {"replicas": 0}  # Prevents running this builder service
}

for i in range(NUM_NODES):
    port = BASE_PORT + i
    name = f"node_{i}"
    services[name] = {
        "image": IMAGE_NAME,
        "command": ["start_node", str(port)],
        "ports": [f"{port}:{port}/udp"]
    }

compose = {
    "services": services
}

with open("docker-compose.yml", "w") as f:
    yaml.dump(compose, f, sort_keys=False)