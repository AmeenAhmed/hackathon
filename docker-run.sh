#!/bin/bash

# Docker Compose convenience script for WayWar

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored output
print_color() {
    printf "${2}${1}${NC}\n"
}

# Function to check if Docker is running
check_docker() {
    if ! docker info > /dev/null 2>&1; then
        print_color "Docker is not running. Please start Docker first." "$RED"
        exit 1
    fi
}

# Function to show usage
show_usage() {
    echo "Usage: $0 [COMMAND]"
    echo ""
    echo "Commands:"
    echo "  start       Start all services in production mode"
    echo "  start-dev   Start all services in development mode with hot-reload"
    echo "  stop        Stop all running services"
    echo "  restart     Restart all services"
    echo "  build       Build all Docker images"
    echo "  rebuild     Rebuild all Docker images (no cache)"
    echo "  logs        Show logs for all services"
    echo "  logs-f      Follow logs for all services"
    echo "  status      Show status of all services"
    echo "  clean       Stop and remove all containers, networks, and volumes"
    echo "  help        Show this help message"
    echo ""
}

# Check Docker is running
check_docker

# Parse command
case "$1" in
    start)
        print_color "Starting WayWar in production mode..." "$GREEN"
        docker-compose up -d
        print_color "WayWar is running!" "$GREEN"
        print_color "Client: http://localhost" "$YELLOW"
        print_color "Server WebSocket: ws://localhost/ws" "$YELLOW"
        ;;

    start-dev)
        print_color "Starting WayWar in development mode..." "$GREEN"
        # Check if dev services exist in docker-compose.yml
        if docker-compose config --services | grep -q "server-dev"; then
            docker-compose up server-dev client-dev
        else
            print_color "Dev services not found. Using production services..." "$YELLOW"
            docker-compose up
        fi
        ;;

    stop)
        print_color "Stopping WayWar..." "$YELLOW"
        docker-compose down
        print_color "WayWar stopped." "$GREEN"
        ;;

    restart)
        print_color "Restarting WayWar..." "$YELLOW"
        docker-compose restart
        print_color "WayWar restarted." "$GREEN"
        ;;

    build)
        print_color "Building WayWar Docker images..." "$GREEN"
        docker-compose build
        print_color "Build complete!" "$GREEN"
        ;;

    rebuild)
        print_color "Rebuilding WayWar Docker images (no cache)..." "$GREEN"
        docker-compose build --no-cache
        print_color "Rebuild complete!" "$GREEN"
        ;;

    logs)
        docker-compose logs
        ;;

    logs-f)
        docker-compose logs -f
        ;;

    status)
        print_color "WayWar services status:" "$GREEN"
        docker-compose ps
        ;;

    clean)
        print_color "Cleaning up WayWar..." "$YELLOW"
        docker-compose down -v --rmi local
        print_color "Cleanup complete!" "$GREEN"
        ;;

    help|--help|-h)
        show_usage
        ;;

    *)
        print_color "Invalid command: $1" "$RED"
        echo ""
        show_usage
        exit 1
        ;;
esac