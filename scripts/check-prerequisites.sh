#!/bin/bash
# Check all required tools before setup

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "Checking prerequisites..."
echo ""

MISSING=0

check_command() {
    if command -v "$1" >/dev/null 2>&1; then
        echo -e "${GREEN}✓${NC} $1"
    else
        echo -e "${RED}✗${NC} $1 (required)"
        MISSING=1
    fi
}

# Check required commands
check_command docker
check_command docker-compose
check_command go
check_command jq
check_command psql
check_command make

echo ""

# Check config.yaml
if [ -f config.yaml ]; then
    echo -e "${GREEN}✓${NC} config.yaml"
else
    echo -e "${RED}✗${NC} config.yaml not found"
    echo -e "${YELLOW}  Run: cp config.example.yaml config.yaml${NC}"
    MISSING=1
fi

echo ""

if [ $MISSING -eq 1 ]; then
    echo -e "${RED}Missing prerequisites. Please install required tools.${NC}"
    exit 1
fi

echo -e "${GREEN}✓ All prerequisites met${NC}"
