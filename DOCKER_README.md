# Docker Setup for WayArena Game

This project includes Docker configurations for both development and production environments.

## Prerequisites

- Docker Engine 20.10+
- Docker Compose 2.0+

## Quick Start

### Production Build

Run the entire application in production mode:

```bash
# Build and start all services
docker-compose up -d

# View logs
docker-compose logs -f

# Stop all services
docker-compose down
```

The application will be available at:
- **Client**: http://localhost:8091
- **Server WebSocket**: ws://localhost:8090/ws (or proxied through nginx at ws://localhost:8091/ws)

### Development Mode

For development with hot-reload:

1. Create a `docker-compose.dev.yml` file or uncomment the dev services in `docker-compose.yml`

2. Run development services:
```bash
# Using the dev services (if uncommented)
docker-compose up server-dev client-dev

# Or create override file
docker-compose -f docker-compose.yml -f docker-compose.dev.yml up
```

## Architecture

### Services

1. **server**: Go backend WebSocket server
   - Port: 8090 (external) -> 8080 (internal)
   - Handles game logic and real-time communication
   - Health check endpoint: `/health`

2. **client**: Vue.js frontend with Phaser game engine
   - Port: 8091 (external) -> 80 (internal)
   - Served via nginx
   - WebSocket proxy to backend server

### Network

- All services communicate through `wayarena-network` (bridge network)
- nginx proxies WebSocket connections from `/ws` to the backend server

## Building Images

### Build individual images:

```bash
# Build server
docker build -t wayarena-server ./server

# Build client
docker build -t wayarena-client ./client
```

### Build with Docker Compose:

```bash
# Build all services
docker-compose build

# Build specific service
docker-compose build server
docker-compose build client
```

## Environment Variables

### Server
- `PORT`: Server port (default: 8080)
- `ENV`: Environment (development/production)

### Client
- `NODE_ENV`: Node environment
- `VITE_WS_URL`: WebSocket URL (auto-configured in production)

## Debugging

### View logs:
```bash
# All services
docker-compose logs -f

# Specific service
docker-compose logs -f server
docker-compose logs -f client
```

### Execute commands in containers:
```bash
# Access server shell
docker-compose exec server sh

# Access client shell
docker-compose exec client sh
```

### Check health status:
```bash
# Check all services
docker-compose ps

# Health check details
docker inspect wayarena-server | grep -A 10 Health
docker inspect wayarena-client | grep -A 10 Health
```

## Volume Mounts (Development)

For development, volumes are mounted to enable hot-reload:

- Server: `./server:/app`
- Client: `./client:/app` (excludes node_modules)

## Troubleshooting

### WebSocket connection issues:
1. Ensure the server is running: `docker-compose ps server`
2. Check nginx proxy configuration in `client/nginx.conf`
3. Verify WebSocket URL in browser console

### Build failures:
1. Clear Docker cache: `docker-compose build --no-cache`
2. Remove volumes: `docker-compose down -v`
3. Check Docker disk space: `docker system df`

### Port conflicts:
1. Current configuration uses:
   - Client: 8091 (external) -> 80 (internal)
   - Server: 8090 (external) -> 8080 (internal)
2. To change ports, modify in `docker-compose.yml`:
   ```yaml
   ports:
     - "9091:80"   # Client on different port
     - "9090:8080" # Server on different port
   ```

## Production Deployment

For production deployment:

1. Update environment variables in `docker-compose.yml`
2. Configure proper domain and SSL certificates
3. Use Docker Swarm or Kubernetes for orchestration
4. Set up monitoring and logging (Prometheus, Grafana, ELK stack)

## Clean Up

Remove all containers, networks, and volumes:

```bash
# Stop and remove containers
docker-compose down

# Remove with volumes
docker-compose down -v

# Remove all including images
docker-compose down --rmi all -v
```