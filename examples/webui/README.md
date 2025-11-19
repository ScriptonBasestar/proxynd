# ProxyND Web Administration Panel

A single-page web application for managing ProxyND through its admin APIs.

## Overview

This WebUI provides a browser-based interface for:
- Package manager enable/disable toggles
- Configuration reload
- Cache management
- Plugin health monitoring
- Real-time status updates

## Quick Start

### 1. Start ProxyND Server

```bash
# From project root
make start
```

The server will start on `http://localhost:8080` by default.

### 2. Open the Web Interface

```bash
# Open in your default browser
open examples/webui/admin-api.html

# Or on Linux
xdg-open examples/webui/admin-api.html

# Or simply drag the file into your browser
```

### 3. Login

**Option A: Development Mode (Skip Login)**
- Click "Skip Login (Dev Mode)" button
- All features work without authentication
- Use for local development only

**Option B: Production Mode (JWT Authentication)**
- Generate a JWT token (see below)
- Enter username and paste token
- Click "Login"

## Generating JWT Tokens

### Using proxyndctl

```bash
# Generate token for admin user
proxyndctl token generate --username admin --role admin

# Generate token with custom expiration
proxyndctl token generate --username admin --role admin --expires 24h
```

### Using curl (direct API call)

```bash
# Login to get JWT token
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin",
    "password": "your-password"
  }'

# Response:
# {
#   "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
#   "expires_at": "2024-01-20T10:30:00Z"
# }
```

### Token Storage

The WebUI stores your JWT token in browser's localStorage for convenience. The token persists across browser sessions until:
- You click "Logout"
- The token expires
- You clear browser storage

## Features

### Package Manager Toggles

Enable or disable package managers dynamically:
- Maven
- NPM
- Docker Registry
- PyPI
- APT
- YUM
- APK

Status indicators:
- 🟢 Green: Enabled
- 🔴 Red: Disabled

### Configuration Reload

Reload server configuration without restart:
1. Edit `config.yaml`
2. Click "Reload Config" button
3. Server picks up changes immediately

**Note**: Some settings require server restart (port changes, TLS certificates).

### Cache Management

**View Statistics**:
- Cache hit rate
- Total requests
- Cache size

**Clear Cache**:
- "Clear All Cache" - Removes all cached packages (requires confirmation)
- "Clear by Type" - Remove cache for specific package manager

### Plugin Health Dashboard

Real-time monitoring of plugin system:
- Plugin status (ready/initializing/failed)
- Error counts
- Initialization timing
- Auto-refresh every 5 seconds (toggleable)

**Status Colors**:
- White background: Healthy plugin
- Red background: Failed plugin with errors

### Activity Log

All API interactions are logged with timestamps:
- Successful operations (green checkmark)
- Failed operations (red X)
- Clear log button to reset view

## API Endpoints Used

The WebUI interacts with these ProxyND APIs:

### Public Endpoints (No Authentication)
- `GET /api/cache/stats` - Cache statistics
- `GET /api/cache/list` - List cached items
- `GET /api/v1/pm` - List package managers
- `GET /api/v1/plugins/health` - Plugin health status

### Protected Endpoints (Requires JWT)
- `POST /api/v1/pm/{name}/toggle` - Enable/disable PM (admin only)
- `POST /api/v1/config/reload` - Reload configuration (admin only)
- `DELETE /api/cache/clear` - Clear all cache (admin only)
- `DELETE /api/cache/clear/{type}` - Clear cache by type (admin only)

## Security Considerations

### Development Environment
- Skip Login mode bypasses all authentication
- Safe for local development only
- **Never use in production**

### Production Environment
- Always use HTTPS (not HTTP)
- Configure JWT secret in config.yaml
- Use strong passwords for user accounts
- Set appropriate token expiration times
- Enable rate limiting (default: 10 req/min per IP)

### HTTPS Setup

```yaml
# config.yaml
server:
  port: 8443
  tls:
    enabled: true
    cert_file: /path/to/cert.pem
    key_file: /path/to/key.pem
```

Then access via: `https://localhost:8443`

### Serving the WebUI

**Option 1: File Protocol (Simple)**
```bash
# Open directly from filesystem
open examples/webui/admin-api.html
```

**Option 2: HTTP Server (Recommended for Production)**
```bash
# Using Python
cd examples/webui
python3 -m http.server 8000

# Using Node.js
npx http-server -p 8000

# Using nginx
# Add location block to serve static files
```

Then access via: `http://localhost:8000/admin-api.html`

## Rate Limiting

All protected endpoints have rate limiting:
- **Limit**: 10 requests per minute per IP
- **Burst**: 3 additional requests allowed

If you exceed the limit:
- HTTP 429 status returned
- Toast notification: "Rate limit exceeded. Please wait..."
- Retry after 60 seconds

## Error Handling

The WebUI handles common API errors:

| Status | Meaning | Action |
|--------|---------|--------|
| 401 | Unauthorized | Auto-logout, redirect to login |
| 403 | Forbidden | Show "Permission denied" toast |
| 429 | Rate Limited | Show rate limit message, wait 60s |
| 500 | Server Error | Show error message in activity log |

## Browser Compatibility

Tested on:
- Chrome 90+
- Firefox 88+
- Safari 14+
- Edge 90+

**Requirements**:
- Modern browser with ES6 support
- JavaScript enabled
- localStorage enabled

## Troubleshooting

### Token expires immediately
**Problem**: JWT token expiration set too short
**Solution**: Generate token with longer expiration:
```bash
proxyndctl token generate --username admin --role admin --expires 24h
```

### "Plugin system not available" error
**Problem**: Plugin manager not initialized
**Solution**: Check ProxyND logs:
```bash
tail -f /var/log/proxynd/proxynd.log
```

Enable plugin system in config:
```yaml
plugins:
  enabled: true
```

### CORS errors when served from different domain
**Problem**: Browser blocks cross-origin requests
**Solution**: Configure CORS in ProxyND:
```yaml
server:
  cors:
    enabled: true
    allowed_origins:
      - "http://localhost:8000"
      - "https://admin.yourcompany.com"
```

### Auto-refresh stops working
**Problem**: Browser tab suspended or connection lost
**Solution**:
1. Click "Refresh Now" button manually
2. Check browser console for errors
3. Verify ProxyND server is running

## Development Mode vs Production Mode

### Development Mode (Skip Login)
```javascript
// In admin-api.html, JWT_TOKEN is set to null
// API calls made without Authorization header
// Server must be configured to allow anonymous access
```

```yaml
# config.yaml - Development
security:
  authentication:
    enabled: false  # Disable auth for dev
```

### Production Mode (JWT Required)
```javascript
// User must login to get valid JWT token
// Token stored in localStorage
// All protected API calls include: Authorization: Bearer <token>
```

```yaml
# config.yaml - Production
security:
  authentication:
    enabled: true
    jwt:
      secret: "your-secret-key-change-this"
      expiration: "1h"
```

## Monitoring Best Practices

1. **Enable Auto-Refresh**: Keep plugin health auto-refresh enabled to catch issues quickly
2. **Watch Activity Log**: Monitor for failed API calls or permission errors
3. **Check Cache Hit Rate**: Low hit rates may indicate cache configuration issues
4. **Monitor Plugin Errors**: Failed plugins appear with red background - investigate immediately

## API Documentation

For detailed API documentation, see:
- `/docs/API_PM_TOGGLE_AUTHENTICATION.md` - Package manager toggle API
- `/docs/API_CONFIG_RELOAD_AUTHENTICATION.md` - Configuration reload API
- `/docs/API_CACHE_CLEAR_AUTHENTICATION.md` - Cache management API
- `/docs/PLUGIN_OPERATIONS_GUIDE.md` - Plugin system operations

## Advanced Usage

### Custom API Base URL

If ProxyND runs on different host/port, edit the HTML:

```javascript
// Change this line in admin-api.html
const API_BASE = 'http://your-server:8080';
```

### Embedding in Application

The WebUI can be embedded in larger applications:

```html
<iframe src="/admin/admin-api.html" width="100%" height="800"></iframe>
```

### Customizing Appearance

The UI uses Tailwind CSS via CDN. To customize:

1. Add custom CSS classes
2. Modify color scheme in HTML
3. Or include custom stylesheet:

```html
<link rel="stylesheet" href="custom-styles.css">
```

## License

Same license as ProxyND (see project root LICENSE file).

## Support

For issues or questions:
- Check ProxyND documentation: `/docs/`
- Review server logs: `/var/log/proxynd/`
- Open GitHub issue: https://github.com/scriptonbasestar/proxynd/issues
