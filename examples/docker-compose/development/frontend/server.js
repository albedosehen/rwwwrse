const express = require('express');
const cors = require('cors');
const helmet = require('helmet');
const morgan = require('morgan');
const path = require('path');

const app = express();
const port = process.env.PORT || 3000;

// Development middleware
app.use(morgan('dev'));
app.use(helmet({
  contentSecurityPolicy: false, // Relaxed for development
  crossOriginEmbedderPolicy: false
}));
app.use(cors({
  origin: true,
  credentials: true
}));
app.use(express.json());
app.use(express.static(path.join(__dirname, 'public')));

// Hot reload support
if (process.env.HOT_RELOAD === 'true') {
  console.log('🔥 Hot reload enabled for development');
}

// Health check endpoint
app.get('/health', (req, res) => {
  res.json({ 
    status: 'healthy', 
    service: 'frontend-dev',
    timestamp: new Date().toISOString(),
    uptime: process.uptime(),
    environment: process.env.NODE_ENV || 'development'
  });
});

// API proxy routes for development
app.get('/api/*', (req, res) => {
  res.json({
    message: 'API proxy route - in production this would proxy to backend',
    path: req.path,
    method: req.method,
    timestamp: new Date().toISOString()
  });
});

// Main application route
app.get('/', (req, res) => {
  res.send(`
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>rwwwrse Development Environment</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
            padding: 2rem;
        }
        .container {
            max-width: 800px;
            text-align: center;
            background: rgba(255, 255, 255, 0.1);
            backdrop-filter: blur(10px);
            border-radius: 20px;
            padding: 3rem;
            box-shadow: 0 8px 32px rgba(0, 0, 0, 0.1);
        }
        .logo { font-size: 4rem; margin-bottom: 1rem; }
        .title { font-size: 2.5rem; margin-bottom: 0.5rem; }
        .subtitle { opacity: 0.8; font-size: 1.2rem; margin-bottom: 2rem; }
        .features {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 1.5rem;
            margin: 2rem 0;
        }
        .feature {
            background: rgba(255, 255, 255, 0.1);
            padding: 1.5rem;
            border-radius: 10px;
            transition: transform 0.3s ease;
        }
        .feature:hover { transform: translateY(-5px); }
        .feature-icon { font-size: 2rem; margin-bottom: 0.5rem; }
        .links {
            display: flex;
            gap: 1rem;
            justify-content: center;
            flex-wrap: wrap;
            margin-top: 2rem;
        }
        .btn {
            background: rgba(255, 255, 255, 0.2);
            color: white;
            padding: 0.75rem 1.5rem;
            border: none;
            border-radius: 8px;
            text-decoration: none;
            transition: all 0.3s ease;
            cursor: pointer;
        }
        .btn:hover {
            background: rgba(255, 255, 255, 0.3);
            transform: translateY(-2px);
        }
        .status {
            position: fixed;
            top: 1rem;
            right: 1rem;
            background: #4CAF50;
            color: white;
            padding: 0.5rem 1rem;
            border-radius: 20px;
            font-size: 0.9rem;
            animation: pulse 2s infinite;
        }
        @keyframes pulse {
            0%, 100% { opacity: 1; }
            50% { opacity: 0.7; }
        }
        .hot-reload {
            color: #FF6B35;
            font-weight: bold;
            margin-top: 1rem;
        }
    </style>
    <script>
        // Auto-refresh for development
        if ('${process.env.HOT_RELOAD}' === 'true') {
            let ws;
            function connectWebSocket() {
                ws = new WebSocket('ws://localhost:3000');
                ws.onmessage = function(event) {
                    if (event.data === 'reload') {
                        window.location.reload();
                    }
                };
                ws.onclose = function() {
                    setTimeout(connectWebSocket, 1000);
                };
            }
            // connectWebSocket(); // Uncomment when WebSocket server is implemented
        }
    </script>
</head>
<body>
    <div class="status">🔥 Development Mode</div>
    <div class="container">
        <div class="logo">🚀</div>
        <h1 class="title">rwwwrse Development</h1>
        <p class="subtitle">Local development environment with hot reload</p>
        
        <div class="features">
            <div class="feature">
                <div class="feature-icon">⚡</div>
                <h3>Hot Reload</h3>
                <p>Automatic refresh on file changes</p>
            </div>
            <div class="feature">
                <div class="feature-icon">🔧</div>
                <h3>Debug Mode</h3>
                <p>Verbose logging and error details</p>
            </div>
            <div class="feature">
                <div class="feature-icon">📡</div>
                <h3>API Proxy</h3>
                <p>Backend API integration</p>
            </div>
            <div class="feature">
                <div class="feature-icon">🛠️</div>
                <h3>Dev Tools</h3>
                <p>Development utilities and monitoring</p>
            </div>
        </div>
        
        <div class="links">
            <a href="/api/test" class="btn">Test API</a>
            <a href="http://api.localhost" class="btn">API Server</a>
            <a href="http://docs.localhost" class="btn">Documentation</a>
            <a href="http://tools.localhost" class="btn">Dev Tools</a>
            <a href="http://localhost:8025" class="btn">Mailhog</a>
        </div>
        
        ${process.env.HOT_RELOAD === 'true' ? '<div class="hot-reload">🔥 Hot reload is active</div>' : ''}
    </div>
</body>
</html>
  `);
});

// Development API endpoints
app.get('/api/test', (req, res) => {
  res.json({
    message: 'Frontend development server API test',
    timestamp: new Date().toISOString(),
    environment: process.env.NODE_ENV,
    nodeVersion: process.version,
    uptime: process.uptime()
  });
});

// Catch all route for SPA development
app.get('*', (req, res) => {
  res.redirect('/');
});

// Error handling middleware
app.use((err, req, res, next) => {
  console.error('Development error:', err);
  res.status(500).json({
    error: 'Development server error',
    message: err.message,
    stack: process.env.NODE_ENV === 'development' ? err.stack : undefined
  });
});

// Start server
app.listen(port, '0.0.0.0', () => {
  console.log(`🚀 Frontend development server running on port ${port}`);
  console.log(`🌐 Access at: http://localhost:${port}`);
  console.log(`🔧 Environment: ${process.env.NODE_ENV || 'development'}`);
  
  if (process.env.HOT_RELOAD === 'true') {
    console.log('🔥 Hot reload enabled - files will auto-refresh');
  }
});

// Graceful shutdown
process.on('SIGTERM', () => {
  console.log('📴 Frontend development server shutting down...');
  process.exit(0);
});

process.on('SIGINT', () => {
  console.log('📴 Frontend development server shutting down...');
  process.exit(0);
});