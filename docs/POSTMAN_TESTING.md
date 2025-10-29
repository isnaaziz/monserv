# Testing MonServ API dengan Postman

## Setup

1. **Install Postman**: Download dari https://www.postman.com/downloads/
2. **Start Server**:
   ```bash
   cd /Users/macbookairm3/Documents/Isna\ Azis\ Nurohman/monserv
   go run cmd/server/main.go
   ```
3. **Verify Server Running**: Check log untuk `Server starting on :18904`

## REST API Testing

### 1. Get All Servers

**Request:**

```
GET http://localhost:18904/api/v1/servers
```

**Headers:**

```
Accept: application/json
```

**Expected Response:**

```json
{
  "success": true,
  "data": {
    "servers": [
      {
        "url": "ssh://scada:***@192.168.4.3:2222",
        "status": "online",
        "metrics": {
          "hostname": "scadanas",
          "uptime_seconds": 3600,
          "memory": {
            "total_bytes": 8589934592,
            "used_bytes": 4294967296,
            "free_bytes": 4294967296,
            "used_percent": 50.0
          },
          "disks": [...],
          "top_processes_by_memory": [...]
        },
        "last_update": "2025-10-29T12:00:00Z"
      }
    ],
    "total": 4
  }
}
```

### 2. Get Server Metrics (dengan Masked URL)

**Request:**

```
GET http://localhost:18904/api/v1/servers/metrics?url=ssh://scada:***@192.168.4.3:2222
```

**Query Parameters:**

- `url`: Server URL (bisa masked atau asli)

**Expected Response:**

```json
{
  "success": true,
  "data": {
    "hostname": "scadanas",
    "uptime_seconds": 3600,
    "memory": {
      "total_bytes": 8589934592,
      "used_bytes": 4294967296,
      "free_bytes": 4294967296,
      "used_percent": 50.0
    },
    "disks": [
      {
        "device": "/dev/sda1",
        "mountpoint": "/",
        "fstype": "ext4",
        "total_bytes": 107374182400,
        "used_bytes": 53687091200,
        "free_bytes": 53687091200,
        "used_percent": 50.0
      }
    ],
    "top_processes_by_memory": [
      {
        "pid": 1234,
        "name": "postgres",
        "username": "postgres",
        "rss_bytes": 536870912,
        "percent_ram": 6.25,
        "cmdline": "postgres -D /var/lib/postgresql/data"
      }
    ],
    "generated_at_utc": "2025-10-29T12:00:00Z"
  }
}
```

### 3. Get Active Alerts

**Request:**

```
GET http://localhost:18904/api/v1/alerts/active
```

**Expected Response:**

```json
{
  "success": true,
  "data": [
    {
      "id": "ssh://scada:***@192.168.4.3:2222|mem",
      "server_url": "ssh://scada:***@192.168.4.3:2222",
      "hostname": "scadanas",
      "type": "memory",
      "severity": "critical",
      "subject": "[ALERT] scadanas memory high",
      "message": "Memory used 85.0% (threshold 80.0%)",
      "is_active": true,
      "triggered_at": "2025-10-29T12:00:00Z"
    }
  ]
}
```

### 4. Get Server Health

**Request:**

```
GET http://localhost:18904/api/v1/health
```

**Expected Response:**

```json
{
  "success": true,
  "data": {
    "status": "ok",
    "servers": {
      "ssh://scada:***@192.168.4.3:2222": {
        "status": "online",
        "hostname": "scadanas"
      },
      "ssh://scada:***@192.168.4.5:2222": {
        "status": "alert",
        "hostname": "scada2"
      }
    },
    "total": 4,
    "online": 3,
    "offline": 1,
    "alerts": 2
  }
}
```

## WebSocket Testing

### Setup WebSocket di Postman

1. **Create New Request**:
   - Click `New` → `WebSocket Request`
2. **Enter URL**:

   ```
   ws://localhost:18904/ws
   ```

3. **Connect**:

   - Click `Connect` button

4. **Monitor Messages**:
   - Tab `Messages` akan menampilkan semua incoming messages

### Expected WebSocket Messages

#### Metrics Update (setiap ~4 detik)

```json
{
  "type": "metrics_update",
  "data": {
    "ssh://scada:***@192.168.4.3:2222": {
      "hostname": "scadanas",
      "uptimeSeconds": 3600,
      "memory": {
        "total": 8589934592,
        "used": 4294967296,
        "free": 4294967296,
        "usedPercent": 50.0
      },
      "disks": [...],
      "topProcsByMem": [...],
      "generatedAtUtc": "2025-10-29T12:00:00Z"
    }
  }
}
```

#### Alert Notification

```json
{
  "type": "alert",
  "alert_type": "alert",
  "subject": "[ALERT] scadanas memory high",
  "message": "Memory used 85.0% (threshold 80.0%)"
}
```

#### Recovery Notification

```json
{
  "type": "alert",
  "alert_type": "recovery",
  "subject": "[RECOVERED] scadanas memory",
  "message": "Memory back to 65.0%"
}
```

## Postman Collection

Anda bisa import collection ini ke Postman:

```json
{
  "info": {
    "name": "MonServ API",
    "schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
  },
  "item": [
    {
      "name": "Get All Servers",
      "request": {
        "method": "GET",
        "header": [],
        "url": {
          "raw": "http://localhost:18904/api/v1/servers",
          "protocol": "http",
          "host": ["localhost"],
          "port": "18904",
          "path": ["api", "v1", "servers"]
        }
      }
    },
    {
      "name": "Get Server Metrics",
      "request": {
        "method": "GET",
        "header": [],
        "url": {
          "raw": "http://localhost:18904/api/v1/servers/metrics?url=ssh://scada:***@192.168.4.3:2222",
          "protocol": "http",
          "host": ["localhost"],
          "port": "18904",
          "path": ["api", "v1", "servers", "metrics"],
          "query": [
            {
              "key": "url",
              "value": "ssh://scada:***@192.168.4.3:2222"
            }
          ]
        }
      }
    },
    {
      "name": "Get Active Alerts",
      "request": {
        "method": "GET",
        "header": [],
        "url": {
          "raw": "http://localhost:18904/api/v1/alerts/active",
          "protocol": "http",
          "host": ["localhost"],
          "port": "18904",
          "path": ["api", "v1", "alerts", "active"]
        }
      }
    },
    {
      "name": "Get Server Health",
      "request": {
        "method": "GET",
        "header": [],
        "url": {
          "raw": "http://localhost:18904/api/v1/health",
          "protocol": "http",
          "host": ["localhost"],
          "port": "18904",
          "path": ["api", "v1", "health"]
        }
      }
    }
  ]
}
```

**Cara Import:**

1. Buka Postman
2. Click `Import` button
3. Pilih `Raw text` tab
4. Paste JSON di atas
5. Click `Import`

## Testing Checklist

### REST API

- [ ] GET /api/v1/servers - Return list servers dengan status
- [ ] GET /api/v1/servers/metrics - Return metrics dengan masked URL
- [ ] GET /api/v1/servers/metrics - Return metrics dengan original URL
- [ ] GET /api/v1/alerts/active - Return active alerts
- [ ] GET /api/v1/health - Return health status semua servers
- [ ] Swagger UI accessible di /swagger/index.html

### WebSocket

- [ ] Connection berhasil ke ws://localhost:18904/ws
- [ ] Receive metrics_update setiap ~4 detik
- [ ] Receive alert notification saat threshold exceeded
- [ ] Receive recovery notification saat kondisi normal
- [ ] Connection auto-reconnect after disconnect
- [ ] Password masked di semua messages

### Error Cases

- [ ] GET /api/v1/servers/metrics?url=invalid - Return 400/404
- [ ] GET /api/v1/servers/metrics tanpa query param - Return 400
- [ ] WebSocket disconnect & reconnect works properly

## Common Issues

### 1. Connection Refused

**Problem**: `Could not connect to http://localhost:18904`

**Solution**:

- Check server is running: `ps aux | grep monserv`
- Check port: `lsof -i :18904`
- Start server: `go run cmd/server/main.go`

### 2. Empty Response

**Problem**: API returns empty data

**Solution**:

- Wait for first polling cycle (~4 seconds)
- Check server logs for SSH connection errors
- Verify agent URLs in .env file

### 3. WebSocket Immediate Disconnect

**Problem**: WebSocket connects then immediately closes

**Solution**:

- Check server logs for errors
- Verify firewall settings
- Try different browser/client

### 4. Masked URL Not Working

**Problem**: GET /api/v1/servers/metrics dengan masked URL return 404

**Solution**:

- Verify URL encoding: `***` harus di-encode sebagai `%2A%2A%2A`
- Copy URL dari `/api/v1/servers` response
- Check server logs untuk actual URL stored

## Performance Testing

### Load Testing dengan Artillery

Install Artillery:

```bash
npm install -g artillery
```

Create test file `load-test.yml`:

```yaml
config:
  target: "http://localhost:18904"
  phases:
    - duration: 60
      arrivalRate: 10
scenarios:
  - name: "Get all servers"
    flow:
      - get:
          url: "/api/v1/servers"
```

Run test:

```bash
artillery run load-test.yml
```

### WebSocket Load Test

```javascript
// ws-load-test.js
const WebSocket = require("ws");

const numConnections = 100;
const connections = [];

for (let i = 0; i < numConnections; i++) {
  const ws = new WebSocket("ws://localhost:18904/ws");

  ws.on("open", () => {
    console.log(`Connection ${i + 1} opened`);
  });

  ws.on("message", (data) => {
    // Count messages received
  });

  connections.push(ws);
}

// Keep connections open
setTimeout(() => {
  console.log(`${connections.length} connections maintained`);
}, 60000);
```

Run:

```bash
node ws-load-test.js
```

## Monitoring

Check server metrics:

```bash
# Connected WebSocket clients
curl http://localhost:18904/api/v1/health | jq '.data'

# Active alerts
curl http://localhost:18904/api/v1/alerts/active | jq '.data | length'

# Server status
curl http://localhost:18904/api/v1/servers | jq '.data.servers[] | {url, status}'
```

## Next Steps

1. Test semua REST endpoints dengan Postman
2. Test WebSocket connection dan messages
3. Verify password masking works correctly
4. Check Swagger UI documentation
5. Monitor server logs during testing
6. Test error cases dan edge cases
