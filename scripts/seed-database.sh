#!/bin/bash

# Seed data script for GitLab Reviewer Roulette Bot
# This script populates the database with test data

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}Starting database seeding...${NC}"

# Check if psql is installed
if ! command -v psql &> /dev/null; then
    echo -e "${RED}Error: psql (PostgreSQL client) is not installed${NC}"
    echo -e "${YELLOW}Install it with:${NC}"
    echo -e "  macOS:        ${GREEN}brew install postgresql@15${NC}"
    echo -e "  Ubuntu/Debian: ${GREEN}sudo apt-get install postgresql-client-15${NC}"
    echo -e "  Arch Linux:   ${GREEN}sudo pacman -S postgresql-libs${NC}"
    exit 1
fi

# Database connection parameters (from environment or defaults)
PGHOST=${POSTGRES_HOST:-localhost}
PGPORT=${POSTGRES_PORT:-5432}
PGDATABASE=${POSTGRES_DB:-reviewer_roulette}
PGUSER=${POSTGRES_USER:-postgres}
PGPASSWORD=${POSTGRES_PASSWORD:-postgres}

export PGPASSWORD

# Function to execute SQL
exec_sql() {
    psql -h "$PGHOST" -p "$PGPORT" -U "$PGUSER" -d "$PGDATABASE" -c "$1" > /dev/null 2>&1
}

# Function to test connection
test_connection() {
    psql -h "$PGHOST" -p "$PGPORT" -U "$PGUSER" -d "$PGDATABASE" -c "SELECT 1;" > /dev/null 2>&1
}

# Check database connection with retry
echo -e "${YELLOW}Checking database connection...${NC}"
MAX_RETRIES=5
RETRY_COUNT=0
while [ $RETRY_COUNT -lt $MAX_RETRIES ]; do
    if test_connection; then
        echo -e "${GREEN}✓ Database connection successful${NC}"
        break
    fi
    RETRY_COUNT=$((RETRY_COUNT + 1))
    if [ $RETRY_COUNT -lt $MAX_RETRIES ]; then
        echo -e "${YELLOW}Waiting for database... (attempt $RETRY_COUNT/$MAX_RETRIES)${NC}"
        sleep 2
    else
        echo -e "${RED}Failed to connect to database after $MAX_RETRIES attempts${NC}"
        echo -e "${YELLOW}Make sure PostgreSQL is running: ${GREEN}docker compose ps postgres${NC}"
        echo -e "${YELLOW}Connection parameters:${NC}"
        echo -e "  Host: $PGHOST"
        echo -e "  Port: $PGPORT"
        echo -e "  Database: $PGDATABASE"
        echo -e "  User: $PGUSER"
        exit 1
    fi
done

# Seed users
echo -e "${YELLOW}Seeding users...${NC}"
exec_sql "
INSERT INTO users (gitlab_id, username, email, role, team, created_at, updated_at) VALUES
(1, 'alice', 'alice@example.com', 'dev', 'team-frontend', NOW(), NOW()),
(2, 'bob', 'bob@example.com', 'dev', 'team-frontend', NOW(), NOW()),
(3, 'charlie', 'charlie@example.com', 'dev', 'team-frontend', NOW(), NOW()),
(4, 'david', 'david@example.com', 'dev', 'team-backend', NOW(), NOW()),
(5, 'eve', 'eve@example.com', 'ops', 'team-backend', NOW(), NOW()),
(6, 'frank', 'frank@example.com', 'dev', 'team-backend', NOW(), NOW()),
(7, 'grace', 'grace@example.com', 'ops', 'team-platform', NOW(), NOW()),
(8, 'henry', 'henry@example.com', 'ops', 'team-platform', NOW(), NOW()),
(9, 'isabel', 'isabel@example.com', 'dev', 'team-mobile', NOW(), NOW()),
(10, 'jack', 'jack@example.com', 'dev', 'team-mobile', NOW(), NOW()),
(11, 'kate', 'kate@example.com', 'dev', 'team-data', NOW(), NOW()),
(12, 'leo', 'leo@example.com', 'ops', 'team-data', NOW(), NOW())
ON CONFLICT (gitlab_id) DO NOTHING;
"
echo -e "${GREEN}✓ Users seeded${NC}"

# Seed badges
echo -e "${YELLOW}Seeding badges...${NC}"
exec_sql "
INSERT INTO badges (name, description, icon, criteria, created_at, updated_at) VALUES
('speed_demon', '⚡ Reviews in less than 2 hours on average', '⚡',
 '{\"metric\": \"avg_ttfr\", \"operator\": \"<\", \"value\": 120}', NOW(), NOW()),
('thorough_reviewer', '🔍 Averages 5+ comments per review', '🔍',
 '{\"metric\": \"avg_comment_count\", \"operator\": \">=\", \"value\": 5}', NOW(), NOW()),
('team_player', '🤝 Most reviews completed this month', '🤝',
 '{\"metric\": \"completed_reviews\", \"operator\": \"top\", \"value\": 1, \"period\": \"month\"}', NOW(), NOW()),
('mentor', '🌟 Most external (cross-team) reviews', '🌟',
 '{\"metric\": \"external_reviews\", \"operator\": \"top\", \"value\": 1, \"period\": \"month\"}', NOW(), NOW())
ON CONFLICT (name) DO NOTHING;
"
echo -e "${GREEN}✓ Badges seeded${NC}"

# Seed some test MR reviews
echo -e "${YELLOW}Seeding test MR reviews...${NC}"
exec_sql "
INSERT INTO mr_reviews (gitlab_mr_iid, gitlab_project_id, mr_url, mr_title, mr_author_id, team,
                        roulette_triggered_at, roulette_triggered_by, status, created_at, updated_at) VALUES
(1, 100, 'https://gitlab.example.com/project/mr/1', 'Add new feature', 1, 'team-frontend',
 NOW() - INTERVAL '2 hours', 2, 'in_review', NOW() - INTERVAL '2 hours', NOW()),
(2, 100, 'https://gitlab.example.com/project/mr/2', 'Fix bug in API', 4, 'team-backend',
 NOW() - INTERVAL '1 day', 5, 'pending', NOW() - INTERVAL '1 day', NOW()),
(3, 101, 'https://gitlab.example.com/project/mr/3', 'Update documentation', 9, 'team-mobile',
 NOW() - INTERVAL '3 hours', 10, 'approved', NOW() - INTERVAL '3 hours', NOW())
ON CONFLICT (gitlab_project_id, gitlab_mr_iid) DO NOTHING;
"
echo -e "${GREEN}✓ Test MR reviews seeded${NC}"

# Seed reviewer assignments
echo -e "${YELLOW}Seeding reviewer assignments...${NC}"
exec_sql "
INSERT INTO reviewer_assignments (mr_review_id, user_id, role, assigned_at, started_review_at) VALUES
(1, 3, 'codeowner', NOW() - INTERVAL '2 hours', NOW() - INTERVAL '1 hour'),
(1, 2, 'team_member', NOW() - INTERVAL '2 hours', NOW() - INTERVAL '1.5 hours'),
(1, 7, 'external', NOW() - INTERVAL '2 hours', NULL),
(2, 6, 'codeowner', NOW() - INTERVAL '1 day', NULL),
(2, 5, 'team_member', NOW() - INTERVAL '1 day', NULL),
(2, 11, 'external', NOW() - INTERVAL '1 day', NULL),
(3, 10, 'codeowner', NOW() - INTERVAL '3 hours', NOW() - INTERVAL '2 hours'),
(3, 9, 'team_member', NOW() - INTERVAL '3 hours', NOW() - INTERVAL '2.5 hours'),
(3, 4, 'external', NOW() - INTERVAL '3 hours', NOW() - INTERVAL '2 hours')
ON CONFLICT DO NOTHING;
"
echo -e "${GREEN}✓ Reviewer assignments seeded${NC}"

# Seed an OOO status (one user on vacation)
echo -e "${YELLOW}Seeding OOO status...${NC}"
exec_sql "
INSERT INTO ooo_status (user_id, start_date, end_date, reason, created_at) VALUES
(8, NOW() - INTERVAL '2 days', NOW() + INTERVAL '5 days', 'Vacation', NOW())
ON CONFLICT DO NOTHING;
"
echo -e "${GREEN}✓ OOO status seeded${NC}"

# Print summary
echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}Database seeding completed successfully!${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo -e "Test users created: 12"
echo -e "  • Frontend team: alice, bob, charlie"
echo -e "  • Backend team: david, eve, frank"
echo -e "  • Platform team: grace, henry (henry is on vacation)"
echo -e "  • Mobile team: isabel, jack"
echo -e "  • Data team: kate, leo"
echo ""
echo -e "Badges created: 4"
echo -e "Test MRs created: 3"
echo -e "Reviewer assignments: 9"
echo ""
echo -e "${YELLOW}Note: These are test GitLab IDs (1-12). In production, these will be replaced with real GitLab user IDs.${NC}"
echo ""
