-- Initialize Mercury Relay database

-- Create events table for analytics
CREATE TABLE IF NOT EXISTS events (
    id VARCHAR(64) PRIMARY KEY,
    pubkey VARCHAR(64) NOT NULL,
    created_at TIMESTAMP NOT NULL,
    kind INTEGER NOT NULL,
    content TEXT,
    sig VARCHAR(128) NOT NULL,
    quality_score FLOAT DEFAULT 0.0,
    is_quarantined BOOLEAN DEFAULT FALSE,
    quarantine_reason TEXT,
    created_at_db TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create blocked npubs table
CREATE TABLE IF NOT EXISTS blocked_npubs (
    npub VARCHAR(64) PRIMARY KEY,
    blocked_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    reason TEXT,
    blocked_by VARCHAR(64)
);

-- Create quality metrics table
CREATE TABLE IF NOT EXISTS quality_metrics (
    id SERIAL PRIMARY KEY,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    total_events INTEGER DEFAULT 0,
    quarantined_events INTEGER DEFAULT 0,
    blocked_npubs INTEGER DEFAULT 0,
    avg_quality_score FLOAT DEFAULT 0.0
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_events_pubkey ON events(pubkey);
CREATE INDEX IF NOT EXISTS idx_events_created_at ON events(created_at);
CREATE INDEX IF NOT EXISTS idx_events_kind ON events(kind);
CREATE INDEX IF NOT EXISTS idx_events_quality_score ON events(quality_score);
CREATE INDEX IF NOT EXISTS idx_events_quarantined ON events(is_quarantined);

-- Create views for analytics
CREATE OR REPLACE VIEW event_stats AS
SELECT 
    DATE(created_at) as date,
    COUNT(*) as total_events,
    COUNT(CASE WHEN is_quarantined THEN 1 END) as quarantined_events,
    AVG(quality_score) as avg_quality_score,
    COUNT(DISTINCT pubkey) as unique_authors
FROM events 
GROUP BY DATE(created_at)
ORDER BY date DESC;

CREATE OR REPLACE VIEW author_stats AS
SELECT 
    pubkey,
    COUNT(*) as event_count,
    AVG(quality_score) as avg_quality_score,
    COUNT(CASE WHEN is_quarantined THEN 1 END) as quarantined_count,
    MAX(created_at) as last_event
FROM events 
GROUP BY pubkey
ORDER BY event_count DESC;

-- Insert initial data
INSERT INTO quality_metrics (total_events, quarantined_events, blocked_npubs, avg_quality_score) 
VALUES (0, 0, 0, 0.0);
