const express = require('express');
const cors = require('cors');
const helmet = require('helmet');
const morgan = require('morgan');
const bodyParser = require('body-parser');
const jwt = require('jsonwebtoken');
const bcrypt = require('bcryptjs');

const app = express();
const port = process.env.PORT || 3000;

// Development middleware
app.use(morgan('combined')); // Detailed logging for development
app.use(helmet({
  contentSecurityPolicy: false, // Relaxed for development
  crossOriginEmbedderPolicy: false
}));
app.use(cors({
  origin: true, // Allow all origins in development
  credentials: true
}));
app.use(bodyParser.json());
app.use(bodyParser.urlencoded({ extended: true }));

// Debug logging middleware
app.use((req, res, next) => {
  console.log(`[${new Date().toISOString()}] ${req.method} ${req.path}`, {
    headers: req.headers,
    body: req.body,
    query: req.query
  });
  next();
});

// Mock database for development
let users = [
  {
    id: 1,
    email: 'admin@example.com',
    password: bcrypt.hashSync('admin123', 10),
    role: 'admin',
    createdAt: new Date().toISOString()
  },
  {
    id: 2,
    email: 'dev@example.com',
    password: bcrypt.hashSync('dev123', 10),
    role: 'developer',
    createdAt: new Date().toISOString()
  }
];

let sessions = [];
let requestLogs = [];

// Middleware to log all requests
app.use((req, res, next) => {
  const logEntry = {
    id: requestLogs.length + 1,
    method: req.method,
    path: req.path,
    timestamp: new Date().toISOString(),
    userAgent: req.get('User-Agent'),
    ip: req.ip
  };
  requestLogs.push(logEntry);
  next();
});

// Health check endpoint
app.get('/health', (req, res) => {
  res.json({
    status: 'healthy',
    service: 'api-dev',
    timestamp: new Date().toISOString(),
    uptime: process.uptime(),
    environment: process.env.NODE_ENV || 'development',
    nodeVersion: process.version,
    memoryUsage: process.memoryUsage(),
    database: {
      status: 'connected', // Mock status
      url: process.env.DATABASE_URL ? 'configured' : 'not configured'
    },
    redis: {
      status: 'connected', // Mock status
      url: process.env.REDIS_URL ? 'configured' : 'not configured'
    }
  });
});

// Authentication endpoints
app.post('/auth/login', async (req, res) => {
  try {
    const { email, password } = req.body;
    
    if (!email || !password) {
      return res.status(400).json({
        error: 'Email and password are required',
        timestamp: new Date().toISOString()
      });
    }

    const user = users.find(u => u.email === email);
    if (!user || !bcrypt.compareSync(password, user.password)) {
      return res.status(401).json({
        error: 'Invalid credentials',
        timestamp: new Date().toISOString()
      });
    }

    const token = jwt.sign(
      { userId: user.id, email: user.email, role: user.role },
      process.env.JWT_SECRET || 'dev-secret',
      { expiresIn: '24h' }
    );

    const session = {
      id: sessions.length + 1,
      userId: user.id,
      token,
      createdAt: new Date().toISOString(),
      expiresAt: new Date(Date.now() + 24 * 60 * 60 * 1000).toISOString()
    };
    sessions.push(session);

    res.json({
      message: 'Login successful',
      token,
      user: {
        id: user.id,
        email: user.email,
        role: user.role
      },
      expiresAt: session.expiresAt,
      timestamp: new Date().toISOString()
    });
  } catch (error) {
    console.error('Login error:', error);
    res.status(500).json({
      error: 'Internal server error',
      timestamp: new Date().toISOString()
    });
  }
});

app.post('/auth/logout', (req, res) => {
  const token = req.headers.authorization?.replace('Bearer ', '');
  
  if (token) {
    const sessionIndex = sessions.findIndex(s => s.token === token);
    if (sessionIndex !== -1) {
      sessions.splice(sessionIndex, 1);
    }
  }

  res.json({
    message: 'Logout successful',
    timestamp: new Date().toISOString()
  });
});

app.get('/auth/validate', (req, res) => {
  const token = req.headers.authorization?.replace('Bearer ', '');
  
  if (!token) {
    return res.status(401).json({
      error: 'No token provided',
      timestamp: new Date().toISOString()
    });
  }

  try {
    const decoded = jwt.verify(token, process.env.JWT_SECRET || 'dev-secret');
    const session = sessions.find(s => s.token === token);
    
    if (!session) {
      return res.status(401).json({
        error: 'Invalid session',
        timestamp: new Date().toISOString()
      });
    }

    res.json({
      valid: true,
      user: decoded,
      session: session,
      timestamp: new Date().toISOString()
    });
  } catch (error) {
    res.status(401).json({
      error: 'Invalid token',
      timestamp: new Date().toISOString()
    });
  }
});

// User management endpoints
app.get('/users', (req, res) => {
  const safeUsers = users.map(({ password, ...user }) => user);
  res.json({
    users: safeUsers,
    total: safeUsers.length,
    timestamp: new Date().toISOString()
  });
});

app.get('/users/:id', (req, res) => {
  const userId = parseInt(req.params.id);
  const user = users.find(u => u.id === userId);
  
  if (!user) {
    return res.status(404).json({
      error: 'User not found',
      timestamp: new Date().toISOString()
    });
  }

  const { password, ...safeUser } = user;
  res.json({
    user: safeUser,
    timestamp: new Date().toISOString()
  });
});

app.post('/users', (req, res) => {
  const { email, password, role = 'user' } = req.body;
  
  if (!email || !password) {
    return res.status(400).json({
      error: 'Email and password are required',
      timestamp: new Date().toISOString()
    });
  }

  if (users.find(u => u.email === email)) {
    return res.status(409).json({
      error: 'User already exists',
      timestamp: new Date().toISOString()
    });
  }

  const newUser = {
    id: users.length + 1,
    email,
    password: bcrypt.hashSync(password, 10),
    role,
    createdAt: new Date().toISOString()
  };

  users.push(newUser);

  const { password: _, ...safeUser } = newUser;
  res.status(201).json({
    message: 'User created successfully',
    user: safeUser,
    timestamp: new Date().toISOString()
  });
});

// API testing endpoints
app.get('/test', (req, res) => {
  res.json({
    message: 'API development server test endpoint',
    timestamp: new Date().toISOString(),
    environment: process.env.NODE_ENV || 'development',
    headers: req.headers,
    query: req.query
  });
});

app.post('/test', (req, res) => {
  res.json({
    message: 'POST test successful',
    receivedData: req.body,
    timestamp: new Date().toISOString()
  });
});

// Development endpoints
app.get('/dev/sessions', (req, res) => {
  res.json({
    sessions: sessions.map(s => ({
      ...s,
      token: `${s.token.substring(0, 20)}...` // Truncate for security
    })),
    total: sessions.length,
    timestamp: new Date().toISOString()
  });
});

app.get('/dev/logs', (req, res) => {
  const limit = parseInt(req.query.limit) || 50;
  const recentLogs = requestLogs.slice(-limit);
  
  res.json({
    logs: recentLogs,
    total: requestLogs.length,
    showing: recentLogs.length,
    timestamp: new Date().toISOString()
  });
});

app.get('/dev/reset', (req, res) => {
  // Reset development data
  sessions.length = 0;
  requestLogs.length = 0;
  
  res.json({
    message: 'Development data reset',
    timestamp: new Date().toISOString()
  });
});

// Mock data endpoints for frontend development
app.get('/api/dashboard', (req, res) => {
  res.json({
    stats: {
      totalUsers: users.length,
      activeSessions: sessions.length,
      totalRequests: requestLogs.length,
      uptime: process.uptime()
    },
    recentActivity: requestLogs.slice(-10),
    timestamp: new Date().toISOString()
  });
});

app.get('/api/metrics', (req, res) => {
  res.json({
    system: {
      memory: process.memoryUsage(),
      uptime: process.uptime(),
      nodeVersion: process.version,
      platform: process.platform
    },
    application: {
      totalUsers: users.length,
      activeSessions: sessions.length,
      totalRequests: requestLogs.length,
      errorRate: 0.02, // Mock
      responseTime: Math.floor(Math.random() * 100) + 20
    },
    timestamp: new Date().toISOString()
  });
});

// Catch-all for undefined routes
app.use('*', (req, res) => {
  res.status(404).json({
    error: 'Endpoint not found',
    path: req.originalUrl,
    method: req.method,
    timestamp: new Date().toISOString(),
    availableEndpoints: [
      'GET /health',
      'POST /auth/login',
      'POST /auth/logout',
      'GET /auth/validate',
      'GET /users',
      'GET /users/:id',
      'POST /users',
      'GET /test',
      'POST /test',
      'GET /dev/sessions',
      'GET /dev/logs',
      'GET /dev/reset',
      'GET /api/dashboard',
      'GET /api/metrics'
    ]
  });
});

// Error handling middleware
app.use((err, req, res, next) => {
  console.error('API Error:', err);
  res.status(err.status || 500).json({
    error: 'Internal server error',
    message: process.env.NODE_ENV === 'development' ? err.message : 'Something went wrong',
    stack: process.env.NODE_ENV === 'development' ? err.stack : undefined,
    timestamp: new Date().toISOString()
  });
});

// Start server
app.listen(port, '0.0.0.0', () => {
  console.log(`🚀 API development server running on port ${port}`);
  console.log(`🌐 Health check: http://localhost:${port}/health`);
  console.log(`🔐 Auth endpoint: http://localhost:${port}/auth/login`);
  console.log(`👥 Users endpoint: http://localhost:${port}/users`);
  console.log(`🧪 Test endpoint: http://localhost:${port}/test`);
  console.log(`🔧 Environment: ${process.env.NODE_ENV || 'development'}`);
  
  if (process.env.DEBUG) {
    console.log('🐛 Debug mode enabled');
  }
});

// Graceful shutdown
process.on('SIGTERM', () => {
  console.log('📴 API development server shutting down...');
  process.exit(0);
});

process.on('SIGINT', () => {
  console.log('📴 API development server shutting down...');
  process.exit(0);
});