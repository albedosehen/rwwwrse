-- Development Database Initialization
-- This script sets up the development database schema

-- Create development users table
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    role VARCHAR(50) DEFAULT 'user',
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create sessions table for development
CREATE TABLE IF NOT EXISTS dev_sessions (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    session_token VARCHAR(255) UNIQUE NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    last_accessed TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create request logs table for development debugging
CREATE TABLE IF NOT EXISTS request_logs (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
    method VARCHAR(10),
    path VARCHAR(255),
    status_code INTEGER,
    response_time_ms INTEGER,
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create development metrics table
CREATE TABLE IF NOT EXISTS dev_metrics (
    id SERIAL PRIMARY KEY,
    metric_name VARCHAR(100) NOT NULL,
    metric_value DECIMAL(15,4),
    metric_unit VARCHAR(20),
    service_name VARCHAR(100),
    recorded_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_dev_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_dev_sessions_token ON dev_sessions(session_token);
CREATE INDEX IF NOT EXISTS idx_dev_sessions_user_id ON dev_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_dev_request_logs_user_id ON request_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_dev_request_logs_created_at ON request_logs(created_at);
CREATE INDEX IF NOT EXISTS idx_dev_metrics_service ON dev_metrics(service_name);
CREATE INDEX IF NOT EXISTS idx_dev_metrics_recorded_at ON dev_metrics(recorded_at);

-- Create a development view for user statistics
CREATE OR REPLACE VIEW dev_user_stats AS
SELECT 
    u.id,
    u.email,
    u.first_name,
    u.last_name,
    u.role,
    u.created_at,
    COUNT(s.id) as active_sessions,
    COUNT(rl.id) as total_requests
FROM users u
LEFT JOIN dev_sessions s ON u.id = s.user_id AND s.expires_at > NOW()
LEFT JOIN request_logs rl ON u.id = rl.user_id
WHERE u.is_active = true
GROUP BY u.id, u.email, u.first_name, u.last_name, u.role, u.created_at;

-- Grant necessary permissions
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO devuser;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO devuser;

-- Create some useful development functions
CREATE OR REPLACE FUNCTION cleanup_old_sessions()
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM dev_sessions WHERE expires_at < NOW();
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION get_user_activity(user_email VARCHAR)
RETURNS TABLE(
    method VARCHAR,
    path VARCHAR,
    status_code INTEGER,
    created_at TIMESTAMP WITH TIME ZONE
) AS $$
BEGIN
    RETURN QUERY
    SELECT rl.method, rl.path, rl.status_code, rl.created_at
    FROM request_logs rl
    JOIN users u ON rl.user_id = u.id
    WHERE u.email = user_email
    ORDER BY rl.created_at DESC
    LIMIT 50;
END;
$$ LANGUAGE plpgsql;

-- Insert initialization completion metric
INSERT INTO dev_metrics (metric_name, metric_value, metric_unit, service_name) VALUES
('database_initialized', 1, 'boolean', 'postgres-dev');