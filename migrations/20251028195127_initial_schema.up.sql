-- Create users table
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    gitlab_id INTEGER UNIQUE NOT NULL,
    username VARCHAR(255) UNIQUE NOT NULL,
    email VARCHAR(255),
    role VARCHAR(50),
    team VARCHAR(100),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_users_gitlab_id ON users(gitlab_id);
CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_team ON users(team);

-- Create ooo_status table
CREATE TABLE IF NOT EXISTS ooo_status (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    start_date TIMESTAMP NOT NULL,
    end_date TIMESTAMP NOT NULL,
    reason TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_ooo_status_user_id ON ooo_status(user_id);
CREATE INDEX idx_ooo_status_dates ON ooo_status(start_date, end_date);

-- Create mr_reviews table
CREATE TABLE IF NOT EXISTS mr_reviews (
    id SERIAL PRIMARY KEY,
    gitlab_mr_iid INTEGER NOT NULL,
    gitlab_project_id INTEGER NOT NULL,
    mr_url TEXT NOT NULL,
    mr_title TEXT,
    mr_author_id INTEGER REFERENCES users(id),
    team VARCHAR(100),
    roulette_triggered_at TIMESTAMP,
    roulette_triggered_by INTEGER REFERENCES users(id),
    first_review_at TIMESTAMP,
    approved_at TIMESTAMP,
    merged_at TIMESTAMP,
    closed_at TIMESTAMP,
    status VARCHAR(50),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(gitlab_project_id, gitlab_mr_iid)
);

CREATE INDEX idx_mr_reviews_status ON mr_reviews(status);
CREATE INDEX idx_mr_reviews_project_mr ON mr_reviews(gitlab_project_id, gitlab_mr_iid);
CREATE INDEX idx_mr_reviews_author ON mr_reviews(mr_author_id);
CREATE INDEX idx_mr_reviews_team ON mr_reviews(team);

-- Create reviewer_assignments table
CREATE TABLE IF NOT EXISTS reviewer_assignments (
    id SERIAL PRIMARY KEY,
    mr_review_id INTEGER NOT NULL REFERENCES mr_reviews(id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR(50),
    assigned_at TIMESTAMP DEFAULT NOW(),
    started_review_at TIMESTAMP,
    first_comment_at TIMESTAMP,
    approved_at TIMESTAMP,
    comment_count INTEGER DEFAULT 0,
    comment_total_length INTEGER DEFAULT 0
);

CREATE INDEX idx_reviewer_assignments_mr_review ON reviewer_assignments(mr_review_id);
CREATE INDEX idx_reviewer_assignments_user ON reviewer_assignments(user_id);
CREATE INDEX idx_reviewer_assignments_role ON reviewer_assignments(role);

-- Create review_metrics table
CREATE TABLE IF NOT EXISTS review_metrics (
    id SERIAL PRIMARY KEY,
    date DATE NOT NULL,
    team VARCHAR(100),
    user_id INTEGER REFERENCES users(id),
    project_id INTEGER,
    total_reviews INTEGER DEFAULT 0,
    completed_reviews INTEGER DEFAULT 0,
    avg_ttfr INTEGER,
    avg_time_to_approval INTEGER,
    avg_comment_count DECIMAL(10,2),
    avg_comment_length DECIMAL(10,2),
    engagement_score DECIMAL(10,2),
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(date, team, user_id, project_id)
);

CREATE INDEX idx_review_metrics_date ON review_metrics(date);
CREATE INDEX idx_review_metrics_team ON review_metrics(team);
CREATE INDEX idx_review_metrics_user ON review_metrics(user_id);
CREATE INDEX idx_review_metrics_project ON review_metrics(project_id);

-- Create badges table
CREATE TABLE IF NOT EXISTS badges (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) UNIQUE NOT NULL,
    description TEXT,
    icon VARCHAR(50),
    criteria JSONB,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_badges_name ON badges(name);

-- Create user_badges table
CREATE TABLE IF NOT EXISTS user_badges (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    badge_id INTEGER NOT NULL REFERENCES badges(id) ON DELETE CASCADE,
    earned_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_user_badges_user ON user_badges(user_id);
CREATE INDEX idx_user_badges_badge ON user_badges(badge_id);
CREATE INDEX idx_user_badges_earned_at ON user_badges(earned_at);

-- Create configuration table
CREATE TABLE IF NOT EXISTS configuration (
    id SERIAL PRIMARY KEY,
    key VARCHAR(255) UNIQUE NOT NULL,
    value JSONB NOT NULL,
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_configuration_key ON configuration(key);
