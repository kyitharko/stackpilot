# stackpilot

Higher-level stack orchestrator. Reads YAML stack files, validates structure, resolves dependency order, and calls the **dockpilot REST API** to deploy or remove services.

**stackpilot never imports the Docker SDK.** All container execution is delegated to dockpilot.

```
User / n8n / AI
      ↓
  stackpilot           ← reads YAML, validates, resolves deps
      ↓ REST API
  dockpilot            ← deploys containers via Docker SDK
      ↓
  Docker Engine
```

## Responsibility

| Concern               | stackpilot | dockpilot |
|-----------------------|:----------:|:---------:|
| Parse YAML            | ✓          |           |
| Validate stack config | ✓          |           |
| Resolve dep graph     | ✓          |           |
| Call Docker SDK       |            | ✓         |
| Pull images           |            | ✓         |
| Manage containers     |            | ✓         |

## Project structure

```
stackpilot/
  main.go
  cmd/
    root.go               — Cobra root
    stack.go              — stack deploy/remove/status/validate subcommands
  internal/
    dockpilot/
      client.go           — Typed HTTP client for the dockpilot REST API
    stack/
      config.go           — Stack, NamedService, ServiceDef types
      parser.go           — Parse / ParseBytes (YAML → Stack)
      validator.go        — Validate (structural checks)
      graph.go            — ResolveOrder (Kahn's algorithm + cycle detection)
      graph_test.go       — Unit tests for dependency resolver
      deployer.go         — Deploy / Remove / Status (calls dockpilot client)
    utils/
      output.go           — Coloured terminal output
  examples/
    backend.yaml          — Example 3-service stack
```

## Stack YAML format

```yaml
name: backend

services:
  mongodb:
    image: mongo:latest
    ports:
      - "27017:27017"
    volumes:
      - "backend-mongodb-data:/data/db"

  redis:
    image: redis:latest
    ports:
      - "6379:6379"

  api:
    image: nginx:alpine
    ports:
      - "8080:80"
    depends_on:
      - mongodb
      - redis
    env:
      - APP_ENV=production
    environment:
      LOG_LEVEL: info
```

Fields:

| Field            | Required | Description                                        |
|------------------|:--------:|----------------------------------------------------|
| `name`           | ✓        | Stack name (top-level)                             |
| `services`       | ✓        | Map of service keys to definitions                 |
| `image`          | ✓        | Docker image reference                             |
| `container_name` |          | Override; defaults to `<stack>-<key>`              |
| `ports`          |          | Port mappings `host:container`                     |
| `volumes`        |          | Named volume mounts `volume:path`                  |
| `env`            |          | Env vars as list `["KEY=VALUE"]`                   |
| `environment`    |          | Env vars as map `{KEY: VALUE}` (merged with `env`) |
| `depends_on`     |          | Services that must deploy before this one          |
| `command`        |          | Override container CMD                             |

## CLI usage

Prerequisites: **dockpilot must be running as an API server.**

```bash
# Start dockpilot API server
dockpilot server --port 8088

# Validate a stack file
stackpilot stack validate examples/backend.yaml

# Deploy a stack
stackpilot stack deploy examples/backend.yaml

# Deploy against a non-default dockpilot instance
stackpilot stack deploy examples/backend.yaml --dockpilot-url http://127.0.0.1:9000

# Show stack status
stackpilot stack status examples/backend.yaml

# Remove a stack
stackpilot stack remove examples/backend.yaml

# Remove a stack and all its named volumes
stackpilot stack remove examples/backend.yaml --volumes
```

## The `--dockpilot-url` flag

Every `stack` subcommand accepts `--dockpilot-url` (default: `http://127.0.0.1:8088`).

```bash
# Run dockpilot on a non-default port
dockpilot server --port 9000
stackpilot stack deploy examples/backend.yaml --dockpilot-url http://127.0.0.1:9000
```

## How deployment works

1. Parse and validate the YAML file.
2. Run `GET /health` on dockpilot — fail fast if unreachable.
3. Compute deployment order (Kahn's topological sort).
4. For each service in order: call `POST /v1/services/{container_name}/deploy`.
5. Removal runs the same list in reverse order.

## Build

```bash
go build -o stackpilot .
./stackpilot --help
```

## Manual test plan

### 1 — Pre-flight

```bash
# Verify dockpilot is reachable
curl http://127.0.0.1:8088/health
# Expected: {"status":"ok"}

# Validate stack before anything touches Docker
stackpilot stack validate examples/backend.yaml
# Expected: "Stack "backend" is valid (3 service(s))"
# Expected deployment order: mongodb → redis → api
```

### 2 — Deploy

```bash
stackpilot stack deploy examples/backend.yaml
# Expected: services deployed in order: mongodb, redis, api
# Expected: no errors

# Verify via dockpilot
curl http://127.0.0.1:8088/v1/services/backend-mongodb/status
curl http://127.0.0.1:8088/v1/services/backend-redis/status
curl http://127.0.0.1:8088/v1/services/backend-api/status
# Expected: state=running for all three
```

### 3 — Idempotency

```bash
stackpilot stack deploy examples/backend.yaml
# Expected: "Already deployed — skipping" for all services (no error)
```

### 4 — Status

```bash
stackpilot stack status examples/backend.yaml
# Expected: table showing SERVICE / CONTAINER / STATE / PORTS
# Expected: all services show "running"
```

### 5 — Remove

```bash
stackpilot stack remove examples/backend.yaml
# Expected: services removed in reverse order: api, redis, mongodb

stackpilot stack status examples/backend.yaml
# Expected: all services show "not deployed"
```

### 6 — Remove with volumes

```bash
stackpilot stack deploy examples/backend.yaml
stackpilot stack remove examples/backend.yaml --volumes
# Expected: containers removed, backend-mongodb-data volume also removed

# Verify volume gone
docker volume ls | grep backend-mongodb-data
# Expected: no output
```

### 7 — Cycle detection

Create `bad.yaml` with a circular dependency and run:
```bash
stackpilot stack validate bad.yaml
# Expected: "circular dependency detected involving: serviceA, serviceB"
```

### 8 — dockpilot unreachable

```bash
stackpilot stack deploy examples/backend.yaml --dockpilot-url http://127.0.0.1:9999
# Expected: "cannot reach dockpilot at http://127.0.0.1:9999: ..."
```
