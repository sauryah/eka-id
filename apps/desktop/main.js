const { app, BrowserWindow, Menu } = require('electron');
const path = require('path');
const fs = require('fs');
const http = require('http');
const { spawn, execSync } = require('child_process');

let mainWindow = null;
let serverProcess = null;
let staticServer = null;
let webPort = 34567;

const MIME_TYPES = {
  '.html': 'text/html; charset=utf-8',
  '.js': 'application/javascript; charset=utf-8',
  '.css': 'text/css; charset=utf-8',
  '.json': 'application/json; charset=utf-8',
  '.png': 'image/png',
  '.jpg': 'image/jpeg',
  '.jpeg': 'image/jpeg',
  '.gif': 'image/gif',
  '.svg': 'image/svg+xml',
  '.ico': 'image/x-icon',
  '.txt': 'text/plain; charset=utf-8',
  '.woff': 'font/woff',
  '.woff2': 'font/woff2',
  '.ttf': 'font/ttf',
};

function getStaticDir() {
  const possiblePaths = [
    path.join(__dirname, 'web', 'out'),
    path.join(process.resourcesPath, 'web', 'out'),
    path.join(__dirname, '..', 'web', 'out'),
  ];
  for (const p of possiblePaths) {
    if (fs.existsSync(p)) {
      return p;
    }
  }
  return path.join(__dirname, 'web', 'out');
}

function getBackendBinaryPath() {
  const possiblePaths = [
    path.join(__dirname, 'resources', 'bin', 'server.exe'),
    path.join(process.resourcesPath, 'bin', 'server.exe'),
    path.join(process.resourcesPath, 'resources', 'bin', 'server.exe'),
    path.join(__dirname, '..', '..', 'services', 'api', 'server.exe'),
  ];
  for (const p of possiblePaths) {
    if (fs.existsSync(p)) {
      return p;
    }
  }
  return path.join(__dirname, 'resources', 'bin', 'server.exe');
}

function startStaticServer(callback) {
  const staticDir = getStaticDir();
  console.log('[Desktop] Serving static assets from:', staticDir);

  staticServer = http.createServer((req, res) => {
    let reqPath = req.url.split('?')[0];
    if (reqPath === '/') {
      reqPath = '/index.html';
    }

    let filePath = path.join(staticDir, reqPath);

    // If file doesn't exist directly, check if adding .html helps (Next.js route convention)
    if (!fs.existsSync(filePath)) {
      const htmlPath = `${filePath}.html`;
      if (fs.existsSync(htmlPath)) {
        filePath = htmlPath;
      } else {
        // Fallback to 404 or index.html for client-side routing
        const notFoundPath = path.join(staticDir, '404.html');
        if (fs.existsSync(notFoundPath)) {
          filePath = notFoundPath;
        } else {
          filePath = path.join(staticDir, 'index.html');
        }
      }
    }

    const ext = path.extname(filePath).toLowerCase();
    const contentType = MIME_TYPES[ext] || 'application/octet-stream';

    fs.readFile(filePath, (err, content) => {
      if (err) {
        res.writeHead(500);
        res.end(`Server Error: ${err.code}`);
        return;
      }
      res.writeHead(200, {
        'Content-Type': contentType,
        'Cache-Control': 'no-cache',
      });
      res.end(content, 'utf-8');
    });
  });

  staticServer.listen(0, '127.0.0.1', () => {
    webPort = staticServer.address().port;
    console.log(`[Desktop] Static Web Server listening on http://127.0.0.1:${webPort}`);
    callback();
  });
}

function checkApiHealth(port) {
  return new Promise((resolve) => {
    const req = http.get(`http://127.0.0.1:${port}/health`, (res) => {
      resolve(res.statusCode === 200);
    });
    req.on('error', () => resolve(false));
    req.setTimeout(1000, () => {
      req.destroy();
      resolve(false);
    });
  });
}

async function startBackend() {
  const isHealthy = await checkApiHealth(8080);
  if (isHealthy) {
    console.log('[Desktop] Go Backend is already running on port 8080.');
    return;
  }

  const binaryPath = getBackendBinaryPath();
  console.log('[Desktop] Spawning Go Backend from:', binaryPath);

  if (!fs.existsSync(binaryPath)) {
    console.warn('[Desktop] Warning: server.exe not found at:', binaryPath);
    return;
  }

  const env = {
    ...process.env,
    ENVIRONMENT: 'development',
    SERVER_PORT: '8080',
    SERVER_HOST: '127.0.0.1',
    DB_HOST: process.env.DB_HOST || '127.0.0.1',
    DB_PORT: process.env.DB_PORT || '5433',
    DB_USER: process.env.DB_USER || 'eka_admin',
    DB_PASSWORD: process.env.DB_PASSWORD || 'eka_secure_dev_pass_2026',
    DB_NAME: process.env.DB_NAME || 'eka_id',
    DB_SSL_MODE: 'disable',
    REDIS_HOST: process.env.REDIS_HOST || '127.0.0.1',
    REDIS_PORT: process.env.REDIS_PORT || '6380',
    JWT_SECRET: 'eka_jwt_dev_super_secret_signing_key_32bytes_minimum_length_2026',
    VERIFY_URL_PREFIX: `http://127.0.0.1:${webPort}/verify`,
    CORS_ALLOWED_ORIGINS: '*',
  };

  try {
    serverProcess = spawn(binaryPath, [], {
      env,
      stdio: ['ignore', 'pipe', 'pipe'],
      detached: false,
      windowsHide: true,
    });

    serverProcess.stdout.on('data', (data) => {
      console.log(`[Backend API] ${data.toString().trim()}`);
    });

    serverProcess.stderr.on('data', (data) => {
      console.error(`[Backend API Err] ${data.toString().trim()}`);
    });

    serverProcess.on('exit', (code, signal) => {
      console.log(`[Backend API] exited with code ${code}, signal ${signal}`);
      serverProcess = null;
    });

    // Wait for API to come online
    let retries = 20;
    while (retries > 0) {
      await new Promise((r) => setTimeout(r, 400));
      const up = await checkApiHealth(8080);
      if (up) {
        console.log('[Desktop] Go Backend is UP and READY.');
        break;
      }
      retries--;
    }
  } catch (err) {
    console.error('[Desktop] Failed to spawn backend process:', err);
  }
}

function createMainWindow() {
  const iconPath = path.join(__dirname, 'build', 'icon.png');

  mainWindow = new BrowserWindow({
    width: 1280,
    height: 860,
    minWidth: 960,
    minHeight: 640,
    title: 'EKA ID - Universal Digital Identity Platform',
    icon: fs.existsSync(iconPath) ? iconPath : undefined,
    webPreferences: {
      preload: path.join(__dirname, 'preload.js'),
      nodeIntegration: false,
      contextIsolation: true,
    },
  });

  Menu.setApplicationMenu(null);

  const startUrl = `http://127.0.0.1:${webPort}`;
  console.log('[Desktop] Loading application from:', startUrl);
  mainWindow.loadURL(startUrl);

  mainWindow.on('closed', () => {
    mainWindow = null;
  });
}

function stopBackend() {
  if (serverProcess) {
    console.log('[Desktop] Terminating background server process...');
    try {
      if (process.platform === 'win32') {
        execSync(`taskkill /pid ${serverProcess.pid} /T /F`);
      } else {
        serverProcess.kill('SIGTERM');
      }
    } catch (e) {
      // Process might already be dead
    }
    serverProcess = null;
  }
  if (staticServer) {
    staticServer.close();
    staticServer = null;
  }
}

app.whenReady().then(() => {
  startStaticServer(async () => {
    await startBackend();
    createMainWindow();

    app.on('activate', () => {
      if (BrowserWindow.getAllWindows().length === 0) {
        createMainWindow();
      }
    });
  });
});

app.on('window-all-closed', () => {
  stopBackend();
  if (process.platform !== 'darwin') {
    app.quit();
  }
});

app.on('before-quit', () => {
  stopBackend();
});
