#!/bin/bash
# =============================================================================
# EventFlow Platform - Cleanup Script
# Removes all containers, volumes, and networks created by the platform
# =============================================================================

set -e

echo "=========================================="
echo "EventFlow Platform - Cleanup"
echo "=========================================="
echo ""

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

# Parse arguments
FORCE=false
VOLUMES=false
IMAGES=false
ALL=false

while [[ $# -gt 0 ]]; do
    case $1 in
        -f|--force)
            FORCE=true
            shift
            ;;
        -v|--volumes)
            VOLUMES=true
            shift
            ;;
        -i|--images)
            IMAGES=true
            shift
            ;;
        -a|--all)
            ALL=true
            VOLUMES=true
            IMAGES=true
            shift
            ;;
        -h|--help)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  -f, --force     Skip confirmation prompts"
            echo "  -v, --volumes   Also remove named volumes (data will be lost)"
            echo "  -i, --images    Also remove built images"
            echo "  -a, --all       Remove everything (volumes + images)"
            echo "  -h, --help      Show this help message"
            echo ""
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

# Confirmation prompt
if [ "$FORCE" = false ]; then
    echo -e "${YELLOW}WARNING: This will stop and remove all EventFlow containers.${NC}"
    
    if [ "$VOLUMES" = true ]; then
        echo -e "${RED}WARNING: Volumes will be removed. ALL DATA WILL BE LOST.${NC}"
    fi
    
    if [ "$IMAGES" = true ]; then
        echo -e "${YELLOW}NOTE: Built images will be removed (they can be rebuilt).${NC}"
    fi
    
    echo ""
    read -p "Are you sure you want to continue? (y/N) " -n 1 -r
    echo
    
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "Aborted."
        exit 0
    fi
fi

cd "$(dirname "$0")/../.."

echo ""
echo "Stopping containers..."
echo "----------------------------------------"
docker-compose down --remove-orphans 2>/dev/null || true

if [ "$VOLUMES" = true ]; then
    echo ""
    echo "Removing volumes..."
    echo "----------------------------------------"
    
    # List of volumes defined in docker-compose.yml
    VOLUME_NAMES=(
        "eventflow-kafka-data"
        "eventflow-zookeeper-data"
        "eventflow-zookeeper-logs"
        "eventflow-redis-data"
        "eventflow-prometheus-data"
        "eventflow-grafana-data"
    )
    
    for vol in "${VOLUME_NAMES[@]}"; do
        full_name="project_${vol}"
        if docker volume inspect "$full_name" >/dev/null 2>&1; then
            docker volume rm "$full_name" 2>/dev/null && \
                echo -e "  ${GREEN}✓${NC} Removed volume: $full_name" || \
                echo -e "  ${YELLOW}!${NC} Could not remove: $full_name"
        fi
    done
fi

if [ "$IMAGES" = true ]; then
    echo ""
    echo "Removing images..."
    echo "----------------------------------------"
    
    # List images built by docker-compose
    IMAGE_NAMES=(
        "project-auth"
        "project-orders"
        "project-payments"
        "project-notification"
        "project-analyzer"
        "project-alert-engine"
        "project-ui-backend"
        "project-dashboard"
    )
    
    for img in "${IMAGE_NAMES[@]}"; do
        if docker image inspect "$img" >/dev/null 2>&1; then
            docker image rm "$img" 2>/dev/null && \
                echo -e "  ${GREEN}✓${NC} Removed image: $img" || \
                echo -e "  ${YELLOW}!${NC} Could not remove: $img"
        fi
    done
fi

echo ""
echo "Removing network..."
echo "----------------------------------------"
docker network rm project_eventflow-network 2>/dev/null && \
    echo -e "  ${GREEN}✓${NC} Removed network: project_eventflow-network" || \
    echo -e "  ${YELLOW}!${NC} Network already removed or not found"

echo ""
echo "=========================================="
echo -e "${GREEN}Cleanup complete!${NC}"
echo ""

if [ "$VOLUMES" = false ]; then
    echo "Note: Volumes were preserved. Use -v flag to remove them."
fi

if [ "$IMAGES" = false ]; then
    echo "Note: Images were preserved. Use -i flag to remove them."
fi
