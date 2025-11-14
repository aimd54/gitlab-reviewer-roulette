#!/bin/bash
# Commit message hook for GitLab Reviewer Roulette Bot
# Validates commit message format (Conventional Commits)
#
# To install: make pre-commit-install
# To skip: git commit --no-verify

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

commit_msg_file=$1
commit_msg=$(cat "$commit_msg_file")

# Skip if message is a merge commit
if echo "$commit_msg" | grep -qE "^Merge (branch|remote-tracking branch)"; then
    exit 0
fi

# Skip if message is a revert commit
if echo "$commit_msg" | grep -qE "^Revert \""; then
    exit 0
fi

# Conventional commits pattern
# Format: type(scope): description
# type: feat, fix, docs, style, refactor, test, chore, perf, ci, build, revert
# scope: optional
# description: required, max 80 chars for first line
pattern="^(feat|fix|docs|style|refactor|test|chore|perf|ci|build|revert)(\(.+\))?: .{1,80}$"

# Get the first line of commit message
first_line=$(echo "$commit_msg" | head -n 1)

if ! echo "$first_line" | grep -qE "$pattern"; then
    echo -e "${RED}✗ Invalid commit message format${NC}"
    echo ""
    echo -e "${YELLOW}Expected format:${NC} ${CYAN}type(scope): description${NC}"
    echo ""
    echo -e "${YELLOW}Types:${NC}"
    echo "  • feat      - New feature"
    echo "  • fix       - Bug fix"
    echo "  • docs      - Documentation changes"
    echo "  • style     - Code style changes (formatting, missing semi-colons, etc.)"
    echo "  • refactor  - Code refactoring (neither fixes a bug nor adds a feature)"
    echo "  • test      - Adding or updating tests"
    echo "  • chore     - Maintenance tasks, dependencies"
    echo "  • perf      - Performance improvements"
    echo "  • ci        - CI/CD changes"
    echo "  • build     - Build system changes (Makefile, Docker, etc.)"
    echo "  • revert    - Revert a previous commit"
    echo ""
    echo -e "${YELLOW}Examples:${NC}"
    echo -e "  ${CYAN}feat(roulette): add expertise-based reviewer selection${NC}"
    echo -e "  ${CYAN}fix(webhook): handle missing CODEOWNERS file${NC}"
    echo -e "  ${CYAN}docs: update README with Docker setup instructions${NC}"
    echo -e "  ${CYAN}refactor(cache): simplify Redis key generation${NC}"
    echo ""
    echo -e "${YELLOW}Your commit message:${NC}"
    echo "  $first_line"
    echo ""
    echo -e "${YELLOW}To bypass this check (not recommended):${NC}"
    echo "  git commit --no-verify"
    echo ""
    exit 1
fi

# Check if first line is too long
if [ ${#first_line} -gt 80 ]; then
    echo -e "${RED}✗ Commit message first line too long${NC}"
    echo ""
    echo -e "${YELLOW}Maximum length:${NC} 80 characters"
    echo -e "${YELLOW}Current length:${NC} ${#first_line} characters"
    echo ""
    echo -e "${YELLOW}Your message:${NC}"
    echo "  $first_line"
    echo ""
    exit 1
fi

echo -e "${GREEN}✓ Commit message format valid${NC}"
exit 0
