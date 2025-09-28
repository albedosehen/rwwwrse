const express = require('express');
const http = require('http');
const socketIo = require('socket.io');
const chokidar = require('chokidar');
const fs = require('fs');
const path = require('path');

const app = express();
const server = http.createServer(app);
const io = socketIo(server, {
  cors: {
    origin: "*",
    methods: ["GET", "POST"]
  }
});

const port = process.env.PORT || 8080;

// Set view engine
app.set('view engine', 'ejs');
app.set('views', path.join(__dirname, 'views'));

// Middleware
app.use(express.json());
app.use(express.static(path.join(__dirname, 'public')));

// Mock system stats
const getSystemStats = () => ({
  uptime: process.uptime(),
  memory: process.memoryUsage(),
  loadAverage: [0.1, 0.2, 0.15], // Mock load average
  cpuUsage: Math.random() * 100,
  diskUsage: {
    total: 100 * 1024 * 1024 * 1024, // 100GB
    used: Math.random() * 50 * 1024 * 1024 * 1024, // Random usage
    free: 50 * 1024 * 1024 * 1024
  },
  timestamp: new Date().toISOString()
});

// Mock container stats
const getContainerStats = () => [
  {
    name: 'rwwwrse-dev',
    status: 'running',
    uptime: process.uptime(),
    cpu: Math.random() * 30,
    memory: Math.random() * 512 * 1024 * 1024,
    network: {
      rx: Math.random() * 1024 * 1024,
      tx: Math.random() * 1024 * 1024
    }
  },
  {
    name: 'frontend-dev',
    status: 'running',
    uptime: process.uptime(),
    cpu: Math.random() * 20,
    memory: Math.random() * 256 * 1024 * 1024,
    network: {
      rx: Math.random() * 512 * 1024,
      tx: Math.random() * 512 * 1024
    }
  },
  {
    name: 'api-dev',
    status: 'running',
    uptime: process.uptime(),
    cpu: Math.random() * 25,
    memory: Math.random() * 256 * 1024 * 1024,
    network: {
      rx: Math.random() * 512 * 1024,
      tx: Math.random() * 512 * 1024
    }
  },
  {
    name: 'postgres-dev',
    status: 'running',
    uptime: process.uptime(),
    cpu: Math.random() * 15,
    memory: Math.random() * 128 * 1024 * 1024,
    network: {
      rx: Math.random() * 256 * 1024,
      tx: Math.random() * 256 * 1024
    }
  }
];

// Store recent logs and metrics
let recentLogs = [];
let metrics = [];
let connectedClients = 0;

// Health check endpoint
app.get('/health', (req, res) => {
  res.json({
    status: 'healthy',
    service: 'dev-tools',
    timestamp: new Date().toISOString(),
    uptime: process.uptime(),
    connectedClients,
    environment: process.env.NODE_ENV || 'development'
  });
});

// Main dashboard
app.get('/', (req, res) => {
  res.render('dashboard', {
    title: 'rwwwrse Development Tools',
    systemStats: getSystemStats(),
    containerStats: getContainerStats(),
    recentLogs: recentLogs.slice(-20),
    timestamp: new Date().toISOString()
  });
});

// API endpoints
app.get('/api/stats', (req, res) => {
  res.json({
    system: getSystemStats(),
    containers: getContainerStats(),
    timestamp: new Date().toISOString()
  });
});

app.get('/api/logs', (req, res) => {
  const limit = parseInt(req.query.limit) || 100;
  res.json({
    logs: recentLogs.slice(-limit),
    total: recentLogs.length,
    timestamp: new Date().toISOString()
  });
});

app.get('/api/metrics', (req, res) => {
  const limit = parseInt(req.query.limit) || 60;
  res.json({
    metrics: metrics.slice(-limit),
    timestamp: new Date().toISOString()
  });
});

app.post('/api/logs', (req, res) => {
  const { level, message, service, timestamp } = req.body;
  
  const logEntry = {
    id: recentLogs.length + 1,
    level: level || 'info',
    message: message || 'No message',
    service: service || 'unknown',
    timestamp: timestamp || new Date().toISOString()
  };
  
  recentLogs.push(logEntry);
  
  // Keep only last 1000 logs
  if (recentLogs.length > 1000) {
    recentLogs = recentLogs.slice(-1000);
  }
  
  // Emit to connected clients
  io.emit('newLog', logEntry);
  
  res.json({ success: true, logEntry });
});

// File watcher endpoints
app.get('/api/watch/start', (req, res) => {
  const watchPath = req.query.path || '/app';
  
  try {
    const watcher = chokidar.watch(watchPath, {
      ignored: /(^|[\/\\])\../, // ignore dotfiles
      persistent: true,
      ignoreInitial: true
    });
    
    watcher.on('change', (filePath) => {
      const changeEvent = {
        type: 'change',
        path: filePath,
        timestamp: new Date().toISOString()
      };
      
      io.emit('fileChange', changeEvent);
      
      const logEntry = {
        id: recentLogs.length + 1,
        level: 'info',
        message: `File changed: ${filePath}`,
        service: 'file-watcher',
        timestamp: new Date().toISOString()
      };
      
      recentLogs.push(logEntry);
      io.emit('newLog', logEntry);
    });
    
    res.json({ 
      success: true, 
      message: `Watching ${watchPath} for changes`,
      watchPath 
    });
  } catch (error) {
    res.status(500).json({ 
      success: false, 
      error: error.message 
    });
  }
});

// Development utilities
app.get('/api/env', (req, res) => {
  // Only show non-sensitive environment variables
  const safeEnv = Object.keys(process.env)
    .filter(key => !key.toLowerCase().includes('secret') && 
                   !key.toLowerCase().includes('password') &&
                   !key.toLowerCase().includes('key'))
    .reduce((obj, key) => {
      obj[key] = process.env[key];
      return obj;
    }, {});
    
  res.json({
    environment: safeEnv,
    nodeVersion: process.version,
    platform: process.platform,
    arch: process.arch,
    timestamp: new Date().toISOString()
  });
});

app.post('/api/restart', (req, res) => {
  const { service } = req.body;
  
  // Mock restart functionality
  const logEntry = {
    id: recentLogs.length + 1,
    level: 'info',
    message: `Restart requested for service: ${service}`,
    service: 'dev-tools',
    timestamp: new Date().toISOString()
  };
  
  recentLogs.push(logEntry);
  io.emit('newLog', logEntry);
  
  res.json({
    success: true,
    message: `Restart signal sent to ${service}`,
    timestamp: new Date().toISOString()
  });
});

// WebSocket handling
io.on('connection', (socket) => {
  connectedClients++;
  console.log(`Client connected. Total clients: ${connectedClients}`);
  
  // Send initial data
  socket.emit('initialData', {
    stats: getSystemStats(),
    containers: getContainerStats(),
    recentLogs: recentLogs.slice(-20)
  });
  
  socket.on('disconnect', () => {
    connectedClients--;
    console.log(`Client disconnected. Total clients: ${connectedClients}`);
  });
  
  socket.on('requestStats', () => {
    socket.emit('statsUpdate', {
      system: getSystemStats(),
      containers: getContainerStats()
    });
  });
});

// Periodic stats updates
setInterval(() => {
  const stats = {
    system: getSystemStats(),
    containers: getContainerStats()
  };
  
  // Store metric
  metrics.push({
    timestamp: new Date().toISOString(),
    ...stats
  });
  
  // Keep only last hour of metrics (assuming 5s intervals)
  if (metrics.length > 720) {
    metrics = metrics.slice(-720);
  }
  
  // Emit to connected clients
  io.emit('statsUpdate', stats);
}, 5000);

// Periodic log generation for demo
setInterval(() => {
  const services = ['rwwwrse', 'frontend-dev', 'api-dev', 'postgres-dev'];
  const levels = ['info', 'debug', 'warn'];
  const messages = [
    'Request processed successfully',
    'Health check passed',
    'Database connection established',
    'Cache hit ratio: 85%',
    'New user session created',
    'File compilation completed'
  ];
  
  if (Math.random() > 0.7) { // 30% chance
    const logEntry = {
      id: recentLogs.length + 1,
      level: levels[Math.floor(Math.random() * levels.length)],
      message: messages[Math.floor(Math.random() * messages.length)],
      service: services[Math.floor(Math.random() * services.length)],
      timestamp: new Date().toISOString()
    };
    
    recentLogs.push(logEntry);
    io.emit('newLog', logEntry);
  }
}, 3000);

// Error handling
app.use((err, req, res, next) => {
  console.error('Dev tools error:', err);
  res.status(500).json({
    error: 'Internal server error',
    message: process.env.NODE_ENV === 'development' ? err.message : 'Something went wrong',
    timestamp: new Date().toISOString()
  });
});

// Start server
server.listen(port, '0.0.0.0', () => {
  console.log(`🛠️ Development tools server running on port ${port}`);
  console.log(`🌐 Dashboard: http://localhost:${port}`);
  console.log(`📊 Stats API: http://localhost:${port}/api/stats`);
  console.log(`📝 Logs API: http://localhost:${port}/api/logs`);
  console.log(`🔧 Environment: ${process.env.NODE_ENV || 'development'}`);
});

// Graceful shutdown
process.on('SIGTERM', () => {
  console.log('📴 Development tools server shutting down...');
  server.close(() => {
    process.exit(0);
  });
});

process.on('SIGINT', () => {
  console.log('📴 Development tools server shutting down...');
  server.close(() => {
    process.exit(0);
  });
});