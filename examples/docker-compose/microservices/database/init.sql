-- Microservices Database Initialization
-- This script sets up the initial database schema and sample data

-- Create users table for authentication service
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

-- Create sessions table for authentication
CREATE TABLE IF NOT EXISTS user_sessions (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    session_token VARCHAR(255) UNIQUE NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    last_accessed TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create API endpoints tracking table
CREATE TABLE IF NOT EXISTS api_endpoints (
    id SERIAL PRIMARY KEY,
    endpoint_path VARCHAR(255) NOT NULL,
    method VARCHAR(10) NOT NULL,
    description TEXT,
    is_public BOOLEAN DEFAULT false,
    rate_limit INTEGER DEFAULT 1000,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create request logs table
CREATE TABLE IF NOT EXISTS request_logs (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
    endpoint_id INTEGER REFERENCES api_endpoints(id) ON DELETE SET NULL,
    ip_address INET,
    user_agent TEXT,
    method VARCHAR(10),
    path VARCHAR(255),
    status_code INTEGER,
    response_time_ms INTEGER,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create notifications table
CREATE TABLE IF NOT EXISTS notifications (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    message TEXT,
    type VARCHAR(50) DEFAULT 'info',
    is_read BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create system metrics table
CREATE TABLE IF NOT EXISTS system_metrics (
    id SERIAL PRIMARY KEY,
    metric_name VARCHAR(100) NOT NULL,
    metric_value DECIMAL(15,4),
    metric_unit VARCHAR(20),
    service_name VARCHAR(100),
    recorded_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Insert sample users
INSERT INTO users (email, password_hash, first_name, last_name, role, is_active) VALUES
('admin@example.com', '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LeaD4XmTq92ztEKmy', 'Admin', 'User', 'admin', true),
('demo@example.com', '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LeaD4XmTq92ztEKmy', 'Demo', 'User', 'user', true),
('api@example.com', '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LeaD4XmTq92ztEKmy', 'API', 'Service', 'service', true),
('test@example.com', '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LeaD4XmTq92ztEKmy', 'Test', 'User', 'user', true);

-- Insert sample API endpoints
INSERT INTO api_endpoints (endpoint_path, method, description, is_public, rate_limit) VALUES
('/api/v1/health', 'GET', 'Health check endpoint', true, 10000),
('/api/v1/auth/login', 'POST', 'User authentication endpoint', true, 100),
('/api/v1/auth/logout', 'POST', 'User logout endpoint', false, 100),
('/api/v1/auth/validate', 'GET', 'Token validation endpoint', false, 1000),
('/api/v1/users', 'GET', 'List users endpoint', false, 500),
('/api/v1/users/{id}', 'GET', 'Get user by ID endpoint', false, 500),
('/api/v1/users', 'POST', 'Create user endpoint', false, 100),
('/api/v1/users/{id}', 'PUT', 'Update user endpoint', false, 100),
('/api/v1/users/{id}', 'DELETE', 'Delete user endpoint', false, 50),
('/api/v1/metrics', 'GET', 'System metrics endpoint', false, 200);

-- Insert sample notifications
INSERT INTO notifications (user_id, title, message, type, is_read) VALUES
(1, 'Welcome to rwwwrse', 'Your microservices architecture is now running successfully!', 'success', false),
(1, 'System Health Check', 'All services are operating normally.', 'info', false),
(2, 'Account Created', 'Your demo account has been created successfully.', 'success', true),
(2, 'New Feature Available', 'Check out the new admin panel functionality.', 'info', false);

-- Insert sample system metrics
INSERT INTO system_metrics (metric_name, metric_value, metric_unit, service_name) VALUES
('cpu_usage', 25.5, 'percent', 'rwwwrse'),
('memory_usage', 512.0, 'MB', 'rwwwrse'),
('request_count', 1247, 'requests', 'api-service'),
('response_time', 45.2, 'ms', 'api-service'),
('active_connections', 12, 'connections', 'auth-service'),
('cache_hit_ratio', 87.3, 'percent', 'redis-cache'),
('database_connections', 8, 'connections', 'postgres-db'),
('queue_size', 3, 'messages', 'message-queue');

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_user_sessions_token ON user_sessions(session_token);
CREATE INDEX IF NOT EXISTS idx_user_sessions_user_id ON user_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_request_logs_user_id ON request_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_request_logs_created_at ON request_logs(created_at);
CREATE INDEX IF NOT EXISTS idx_notifications_user_id ON notifications(user_id);
CREATE INDEX IF NOT EXISTS idx_system_metrics_service ON system_metrics(service_name);
CREATE INDEX IF NOT EXISTS idx_system_metrics_recorded_at ON system_metrics(recorded_at);

-- Create a view for user statistics
CREATE OR REPLACE VIEW user_stats AS
SELECT 
    u.id,
    u.email,
    u.first_name,
    u.last_name,
    u.role,
    u.created_at,
    COUNT(s.id) as active_sessions,
    COUNT(rl.id) as total_requests,
    COUNT(n.id) as unread_notifications
FROM users u
LEFT JOIN user_sessions s ON u.id = s.user_id AND s.expires_at > NOW()
LEFT JOIN request_logs rl ON u.id = rl.user_id
LEFT JOIN notifications n ON u.id = n.user_id AND n.is_read = false
WHERE u.is_active = true
GROUP BY u.id, u.email, u.first_name, u.last_name, u.role, u.created_at;

-- Create a function to clean up expired sessions
CREATE OR REPLACE FUNCTION cleanup_expired_sessions()
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM user_sessions WHERE expires_at < NOW();
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Create a function to update user's last accessed time
CREATE OR REPLACE FUNCTION update_session_access(token VARCHAR(255))
RETURNS BOOLEAN AS $$
BEGIN
    UPDATE user_sessions 
    SET last_accessed = CURRENT_TIMESTAMP 
    WHERE session_token = token AND expires_at > NOW();
    
    RETURN FOUND;
END;
$$ LANGUAGE plpgsql;

-- Insert some sample request logs for demonstration
INSERT INTO request_logs (user_id, endpoint_id, ip_address, user_agent, method, path, status_code, response_time_ms) VALUES
(1, 1, '192.168.1.100', 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36', 'GET', '/api/v1/health', 200, 15),
(2, 2, '192.168.1.101', 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36', 'POST', '/api/v1/auth/login', 200, 87),
(2, 5, '192.168.1.101', 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36', 'GET', '/api/v1/users', 200, 42),
(1, 10, '192.168.1.100', 'curl/7.68.0', 'GET', '/api/v1/metrics', 200, 23);

-- Grant necessary permissions
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO dbuser;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO dbuser;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO dbuser;

-- Insert completion log
INSERT INTO system_metrics (metric_name, metric_value, metric_unit, service_name) VALUES
('database_initialized', 1, 'boolean', 'postgres-db');

-- Display initialization summary
SELECT 
    'Database Initialization Complete' as status,
    COUNT(*) as total_users
FROM users
UNION ALL
SELECT 
    'API Endpoints Configured' as status,
    COUNT(*) as count
FROM api_endpoints
UNION ALL
SELECT 
    'Sample Data Loaded' as status,
    COUNT(*) as count
FROM request_logs;