-- Development Database Seed Data
-- This script populates the development database with sample data

-- Insert development users
INSERT INTO users (email, password_hash, first_name, last_name, role, is_active) VALUES
('admin@example.com', '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LeaD4XmTq92ztEKmy', 'Admin', 'Developer', 'admin', true),
('dev@example.com', '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LeaD4XmTq92ztEKmy', 'Dev', 'User', 'developer', true),
('test@example.com', '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LeaD4XmTq92ztEKmy', 'Test', 'User', 'user', true),
('demo@example.com', '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LeaD4XmTq92ztEKmy', 'Demo', 'Account', 'user', true);

-- Note: All passwords are 'dev123' for development convenience

-- Insert sample request logs for development testing
INSERT INTO request_logs (user_id, method, path, status_code, response_time_ms, ip_address, user_agent) VALUES
(1, 'GET', '/health', 200, 15, '127.0.0.1', 'curl/7.68.0'),
(2, 'POST', '/auth/login', 200, 87, '127.0.0.1', 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)'),
(2, 'GET', '/users', 200, 42, '127.0.0.1', 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)'),
(3, 'GET', '/test', 200, 23, '127.0.0.1', 'PostmanRuntime/7.29.0'),
(1, 'GET', '/dev/logs', 200, 31, '127.0.0.1', 'Mozilla/5.0 (Windows NT 10.0; Win64; x64)'),
(4, 'POST', '/auth/login', 401, 65, '127.0.0.1', 'curl/7.68.0'),
(2, 'GET', '/api/dashboard', 200, 58, '127.0.0.1', 'fetch');

-- Insert sample development metrics
INSERT INTO dev_metrics (metric_name, metric_value, metric_unit, service_name) VALUES
('requests_per_minute', 45.7, 'requests/min', 'api-dev'),
('response_time_avg', 87.3, 'ms', 'api-dev'),
('memory_usage', 234.5, 'MB', 'api-dev'),
('cpu_usage', 12.4, 'percent', 'api-dev'),
('requests_per_minute', 67.2, 'requests/min', 'frontend-dev'),
('response_time_avg', 23.1, 'ms', 'frontend-dev'),
('memory_usage', 156.8, 'MB', 'frontend-dev'),
('cpu_usage', 8.7, 'percent', 'frontend-dev'),
('memory_usage', 512.3, 'MB', 'rwwwrse-dev'),
('cpu_usage', 5.2, 'percent', 'rwwwrse-dev'),
('active_connections', 12, 'connections', 'rwwwrse-dev'),
('proxied_requests', 234, 'requests', 'rwwwrse-dev'),
('database_connections', 3, 'connections', 'postgres-dev'),
('cache_hit_ratio', 89.4, 'percent', 'redis-dev'),
('memory_usage', 67.2, 'MB', 'redis-dev'),
('dev_tools_active_sessions', 2, 'sessions', 'dev-tools'),
('file_changes_detected', 15, 'changes', 'file-watcher'),
('hot_reloads_triggered', 8, 'reloads', 'file-watcher');

-- Insert sample development sessions for testing
INSERT INTO dev_sessions (user_id, session_token, expires_at) VALUES
(1, 'dev_token_admin_12345', NOW() + INTERVAL '24 hours'),
(2, 'dev_token_dev_67890', NOW() + INTERVAL '24 hours'),
(3, 'dev_token_test_abcde', NOW() + INTERVAL '12 hours');

-- Display seed data summary
SELECT 
    'Users Created' as category,
    COUNT(*) as count
FROM users
UNION ALL
SELECT 
    'Request Logs' as category,
    COUNT(*) as count
FROM request_logs
UNION ALL
SELECT 
    'Development Metrics' as category,
    COUNT(*) as count
FROM dev_metrics
UNION ALL
SELECT 
    'Active Sessions' as category,
    COUNT(*) as count
FROM dev_sessions
WHERE expires_at > NOW();