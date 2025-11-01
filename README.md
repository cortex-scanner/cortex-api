# Cortex API Server

## Prerequisites

- Go 1.25 or higher
- Docker (for containerized deployment)
- Task (https://taskfile.dev/) - optional but recommended

## Building the Application

### Using Task (Recommended)

```bash
# Build the application
task build

# Run the application
task run
```

### Using Go Directly

```bash
# Build the application
go build -o build/cortex-api ./cmd/

# Run the application
./build/cortex-api
```

## Running with Docker

### Building the Docker Image

```bash
docker build -t cortex-api .
```