# WebSocket API Documentation

## Overview

MonServ menyediakan WebSocket endpoint untuk real-time monitoring tanpa polling. Server akan otomatis push update metrics dan alerts ke semua connected clients.

## Endpoint

```
ws://localhost:18904/ws
```

## Connection

### JavaScript/Browser Example

```javascript
// Koneksi ke WebSocket
const ws = new WebSocket("ws://localhost:18904/ws");

// Handle connection open
ws.onopen = () => {
  console.log("WebSocket connected");
};

// Handle incoming messages
ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  console.log("Received:", data);

  if (data.type === "metrics_update") {
    // Update UI dengan metrics baru
    updateMetricsUI(data.data);
  } else if (data.type === "alert") {
    // Tampilkan notifikasi alert
    showAlert(data.subject, data.message);
  }
};

// Handle errors
ws.onerror = (error) => {
  console.error("WebSocket error:", error);
};

// Handle connection close
ws.onclose = () => {
  console.log("WebSocket disconnected");
  // Reconnect after 5 seconds
  setTimeout(() => {
    connectWebSocket();
  }, 5000);
};
```

### Python Example

```python
import asyncio
import websockets
import json

async def connect():
    uri = "ws://localhost:18904/ws"
    async with websockets.connect(uri) as websocket:
        print("Connected to WebSocket")

        while True:
            try:
                message = await websocket.recv()
                data = json.loads(message)

                if data['type'] == 'metrics_update':
                    print(f"Metrics update: {len(data['data'])} servers")
                elif data['type'] == 'alert':
                    print(f"Alert: {data['subject']}")

            except websockets.exceptions.ConnectionClosed:
                print("Connection closed")
                break

asyncio.run(connect())
```

### Node.js Example

```javascript
const WebSocket = require("ws");

const ws = new WebSocket("ws://localhost:18904/ws");

ws.on("open", () => {
  console.log("Connected to WebSocket");
});

ws.on("message", (data) => {
  const message = JSON.parse(data);

  if (message.type === "metrics_update") {
    console.log("Metrics update:", Object.keys(message.data).length, "servers");
  } else if (message.type === "alert") {
    console.log("Alert:", message.subject);
  }
});

ws.on("error", (error) => {
  console.error("WebSocket error:", error);
});

ws.on("close", () => {
  console.log("WebSocket closed");
});
```

## Message Types

### 1. Metrics Update

Server mengirim metrics update setiap kali polling selesai (default setiap 4 detik).

**Message Format:**

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
      "disks": [
        {
          "device": "/dev/sda1",
          "mountpoint": "/",
          "fstype": "ext4",
          "total": 107374182400,
          "used": 53687091200,
          "free": 53687091200,
          "usedPercent": 50.0
        }
      ],
      "topProcsByMem": [
        {
          "pid": 1234,
          "name": "postgres",
          "username": "postgres",
          "rssBytes": 536870912,
          "percentRAM": 6.25,
          "cmdline": "postgres -D /var/lib/postgresql/data"
        }
      ],
      "generatedAtUtc": "2025-10-29T12:00:00Z"
    }
  }
}
```

### 2. Alert Notification

Server mengirim alert ketika ada threshold yang terlampaui.

**Message Format:**

```json
{
  "type": "alert",
  "alert_type": "alert",
  "subject": "[ALERT] scadanas memory high",
  "message": "Memory used 85.0% (threshold 80.0%)"
}
```

### 3. Recovery Notification

Server mengirim recovery notification ketika kondisi kembali normal.

**Message Format:**

```json
{
  "type": "alert",
  "alert_type": "recovery",
  "subject": "[RECOVERED] scadanas memory",
  "message": "Memory back to 65.0%"
}
```

## Features

### Auto-Reconnect

WebSocket connection bisa putus karena berbagai alasan (network issues, server restart, dll). Implementasi client harus include auto-reconnect logic:

```javascript
function connectWebSocket() {
  const ws = new WebSocket("ws://localhost:18904/ws");

  ws.onclose = () => {
    console.log("Reconnecting in 5 seconds...");
    setTimeout(connectWebSocket, 5000);
  };

  return ws;
}

const ws = connectWebSocket();
```

### Heartbeat/Ping

Server otomatis mengirim ping setiap 54 detik untuk menjaga connection tetap alive. Client harus respond dengan pong (browser WebSocket API handle ini otomatis).

### Connection Timeout

- **Read Timeout**: 60 detik
- **Write Timeout**: 10 detik
- **Ping Interval**: 54 detik

Jika client tidak respond terhadap ping dalam 60 detik, server akan close connection.

## Testing

### Using wscat (Command Line)

Install wscat:

```bash
npm install -g wscat
```

Connect dan monitor messages:

```bash
wscat -c ws://localhost:18904/ws
```

Output:

```
Connected (press CTRL+C to quit)
< {"type":"metrics_update","data":{...}}
< {"type":"alert","alert_type":"alert","subject":"[ALERT] scadanas memory high","message":"Memory used 85.0% (threshold 80.0%)"}
```

### Using Postman

1. Buka Postman
2. Create New → WebSocket Request
3. Enter URL: `ws://localhost:18904/ws`
4. Click **Connect**
5. Monitor messages di **Messages** tab

### Using Browser Console

```javascript
// Open browser console (F12) dan paste:
const ws = new WebSocket("ws://localhost:18904/ws");
ws.onmessage = (e) => console.log(JSON.parse(e.data));
```

## Performance

### Bandwidth Usage

- **Metrics Update**: ~2-5 KB per server per update
- **Alert**: ~200 bytes per alert
- **Ping/Pong**: ~10 bytes per 54 seconds

Contoh: Monitoring 10 servers dengan polling interval 4 detik:

- Data rate: ~10-12 KB/s
- Bandwidth: ~30-40 MB/jam

### Scalability

- Server dapat handle ratusan concurrent WebSocket connections
- Setiap connection menggunakan ~2-5 MB memory
- Broadcast ke 100 clients: <10ms latency

## Security Considerations

### Production Deployment

1. **Gunakan WSS (WebSocket Secure)**

   ```javascript
   const ws = new WebSocket("wss://your-domain.com/ws");
   ```

2. **Authentication**
   Tambahkan token di query parameter:

   ```javascript
   const ws = new WebSocket("ws://localhost:18904/ws?token=YOUR_AUTH_TOKEN");
   ```

3. **Origin Check**
   Update `CheckOrigin` di `internal/websocket/client.go`:

   ```go
   CheckOrigin: func(r *http.Request) bool {
       origin := r.Header.Get("Origin")
       return origin == "https://your-allowed-domain.com"
   },
   ```

4. **Rate Limiting**
   Batasi jumlah connections per IP

## Error Handling

### Common Errors

1. **Connection Refused**

   - Server belum running
   - Port salah
   - Firewall blocking

2. **Connection Closed Unexpectedly**

   - Network issue
   - Server restart
   - Implement auto-reconnect

3. **Message Parse Error**
   - Invalid JSON dari server
   - Check server logs

### Debugging

Enable verbose logging:

```javascript
ws.onerror = (error) => {
  console.error("WebSocket error:", error);
};

ws.onclose = (event) => {
  console.log("WebSocket closed:", event.code, event.reason);
};
```

## Complete Integration Example

```html
<!DOCTYPE html>
<html>
  <head>
    <title>MonServ WebSocket Monitor</title>
  </head>
  <body>
    <h1>Server Monitor</h1>
    <div id="status">Connecting...</div>
    <div id="metrics"></div>
    <div id="alerts"></div>

    <script>
      let ws;

      function connect() {
        ws = new WebSocket("ws://localhost:18904/ws");

        ws.onopen = () => {
          document.getElementById("status").textContent = "Connected";
          document.getElementById("status").style.color = "green";
        };

        ws.onmessage = (event) => {
          const data = JSON.parse(event.data);

          if (data.type === "metrics_update") {
            updateMetrics(data.data);
          } else if (data.type === "alert") {
            showAlert(data);
          }
        };

        ws.onerror = (error) => {
          console.error("WebSocket error:", error);
        };

        ws.onclose = () => {
          document.getElementById("status").textContent =
            "Disconnected - Reconnecting...";
          document.getElementById("status").style.color = "red";
          setTimeout(connect, 5000);
        };
      }

      function updateMetrics(metrics) {
        const metricsDiv = document.getElementById("metrics");
        let html = "<h2>Server Metrics</h2>";

        for (const [url, data] of Object.entries(metrics)) {
          html += `
                    <div class="server">
                        <h3>${data.hostname}</h3>
                        <p>Memory: ${data.memory.usedPercent.toFixed(1)}%</p>
                        <p>Uptime: ${data.uptimeSeconds}s</p>
                    </div>
                `;
        }

        metricsDiv.innerHTML = html;
      }

      function showAlert(alert) {
        const alertsDiv = document.getElementById("alerts");
        const alertEl = document.createElement("div");
        alertEl.className = alert.alert_type;
        alertEl.innerHTML = `
                <strong>${alert.subject}</strong><br>
                ${alert.message}<br>
                <small>${new Date().toLocaleString()}</small>
            `;
        alertsDiv.insertBefore(alertEl, alertsDiv.firstChild);
      }

      // Start connection
      connect();
    </script>

    <style>
      .server {
        border: 1px solid #ccc;
        padding: 10px;
        margin: 10px 0;
        border-radius: 5px;
      }
      .alert {
        background: #fee;
        border-left: 4px solid red;
        padding: 10px;
        margin: 10px 0;
      }
      .recovery {
        background: #efe;
        border-left: 4px solid green;
        padding: 10px;
        margin: 10px 0;
      }
    </style>
  </body>
</html>
```

## Support

Untuk pertanyaan atau issues, check:

- Server logs untuk debugging
- Network tab di browser DevTools
- WebSocket connection status

## Changelog

- **v1.0** (2025-10-29): Initial WebSocket implementation
  - Real-time metrics broadcast
  - Alert notifications
  - Auto-reconnect support

123456789
