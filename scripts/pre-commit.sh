#!/bin/bash
# Pre-commit hook for GitLab Reviewer Roulette Bot
# Runs fast quality checks before allowing commit
#
# To install: make pre-commit-install
# To skip: git commit --no-verify

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

echo -e "${CYAN}🔍 Running pre-commit checks...${NC}"
echo ""

# Function to print step
print_step() {
    echo -e "${CYAN}▶ $1${NC}"
}

# Function to print success
print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

# Function to print error
print_error() {
    echo -e "${RED}✗ $1${NC}"
}

# Function to print warning
print_warning() {
    echo -e "${YELLOW}⚠ $1${NC}"
}

# Change to repository root
cd "$(git rev-parse --show-toplevel)"

# 1. Check code formatting
print_step "Checking code formatting..."
if ! make fmt-check 2>&1 | grep -v "^make"; then
    print_error "Code formatting issues detected"
    echo ""
    echo "Fix with: make fmt"
    exit 1
fi
print_success "Code is properly formatted"
echo ""

# 2. Run go vet
print_step "Running go vet..."
if ! make vet 2>&1 | grep -v "^make"; then
    print_error "Go vet found issues"
    exit 1
fi
print_success "Go vet passed"
echo ""

# 3. Run linter
print_step "Running golangci-lint..."
if ! make lint 2>&1 | grep -v "^make"; then
    print_error "Linting failed"
    echo ""
    echo "Review the errors above and fix them before committing"
    exit 1
fi
print_success "Linting passed"
echo ""

# 4. Run fast tests (skip race detector for speed)
print_step "Running unit tests (fast mode)..."
if ! make test-short 2>&1 | grep -v "^make"; then
    print_error "Tests failed"
    echo ""
    echo "Fix failing tests before committing"
    exit 1
fi
print_success "Tests passed"
echo ""

# Success!
echo -e "${GREEN}✅ All pre-commit checks passed!${NC}"
echo ""
print_warning "Note: Full tests with race detector will run in CI"
echo ""

exit 0
