# Frontend WebSocket Integration

## Overview

Frontend web UI telah diintegrasikan dengan WebSocket untuk mendapatkan update real-time dari server. Sistem ini menggunakan auto-reconnect dan fallback ke HTTP polling jika WebSocket tidak tersedia.

## Features

### 1. **Real-time Updates**

- Menerima update metrics secara real-time tanpa polling
- Update otomatis setiap ada perubahan di server
- Tidak ada delay 5 detik seperti polling

### 2. **Auto-Reconnect**

- Reconnect otomatis jika koneksi terputus
- Exponential backoff (5s, 10s, 15s, max 30s)
- Fallback ke polling setelah 3 kali gagal

### 3. **Connection Status**

- Visual indicator di halaman (dot dengan warna)
- Status: Connected, Connecting, Disconnected, Error, Polling
- Status text yang informatif

### 4. **Notifications**

- Browser notifications untuk alerts (jika diizinkan)
- In-page notifications dengan animasi
- Auto-dismiss setelah 5 detik

### 5. **Alert Sound**

- Play sound ketika ada alert
- Volume rendah (30%) agar tidak mengganggu
- Graceful fallback jika audio tidak supported

## How It Works

### WebSocket Connection

```javascript
// Koneksi otomatis saat page load
const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
const wsUrl = `${protocol}//${window.location.host}/ws`;
ws = new WebSocket(wsUrl);
```

### Message Handling

Backend mengirim 2 tipe message:

#### 1. Metrics Update

```json
{
  "type": "metrics_update",
  "data": {
    "servers": [...],
    "summary": {...}
  }
}
```

Frontend akan render data ini ke UI.

#### 2. Alert

```json
{
  "type": "alert",
  "alert_type": "alert",
  "subject": "High Memory",
  "message": "Server prod-web-01 memory 95.2% ≥ 90%"
}
```

Frontend akan show notification dan play sound.

### Auto-Reconnect Logic

```javascript
ws.onclose = (event) => {
  connectionAttempts++;
  const delay = Math.min(5000 * connectionAttempts, 30000); // Max 30s

  setTimeout(() => {
    if (useWebSocket) {
      connectWebSocket();
    }
  }, delay);

  // Fallback ke polling setelah 3 kali gagal
  if (connectionAttempts >= 3) {
    useWebSocket = false;
    loop(); // Start polling
  }
};
```

### Fallback to Polling

Jika WebSocket gagal connect 3 kali, sistem otomatis fallback ke HTTP polling:

```javascript
// Polling dengan interval 5 detik
async function loop() {
  const state = await fetchState();
  render(state);
  setTimeout(loop, 5000);
}
```

## Status Indicators

### Status Dot Colors

- 🟢 **Green (Connected)**: WebSocket connected, real-time updates aktif
- 🟡 **Yellow (Connecting)**: Sedang mencoba connect ke WebSocket
- ⚫ **Gray (Disconnected)**: WebSocket terputus, akan reconnect
- 🔴 **Red (Error)**: Error pada koneksi
- 🔵 **Blue (Polling)**: Menggunakan HTTP polling sebagai fallback

### Status Text

- "Real-time Connected" - WebSocket aktif
- "Connecting..." - Mencoba connect
- "Disconnected" - Koneksi terputus
- "Connection Error" - Ada error
- "Using HTTP Polling" - Fallback ke polling

## Notifications

### Browser Notifications

Jika user mengizinkan notifications:

```javascript
// Request permission saat page load
if ("Notification" in window && Notification.permission === "default") {
  Notification.requestPermission();
}

// Show notification
new Notification(title, {
  body: message,
  icon: "/static/icon.png",
  tag: "monserv-notification",
});
```

### In-page Notifications

Notifications muncul di kanan atas dengan animasi slide-in:

```html
<div class="notification notification-success">
  <strong>Connected</strong>
  <p>Real-time updates enabled</p>
</div>
```

Auto-dismiss setelah 5 detik.

## Manual Control

### Toggle Connection Mode

Tersedia function global untuk toggle antara WebSocket dan Polling:

```javascript
// Dari browser console
window.toggleConnectionMode();
```

Ini akan switch antara:

- WebSocket mode (real-time)
- Polling mode (5 detik interval)

## Event Handlers

### Page Visibility

Otomatis reconnect WebSocket ketika tab kembali visible:

```javascript
document.addEventListener("visibilitychange", () => {
  if (!document.hidden && useWebSocket) {
    if (!ws || ws.readyState !== WebSocket.OPEN) {
      connectWebSocket();
    }
  }
});
```

### Page Unload

Cleanup koneksi saat page close:

```javascript
window.addEventListener("beforeunload", () => {
  disconnectWebSocket();
});
```

## Testing

### 1. Test WebSocket Connection

1. Buka browser ke `http://localhost:18904`
2. Check status indicator di kanan atas
3. Harus menunjukkan "Real-time Connected" dengan dot hijau

### 2. Test Real-time Updates

1. Monitor page terbuka
2. Tunggu update dari server (setiap 30 detik)
3. Metrics harus update tanpa refresh page
4. Timestamp harus update otomatis

### 3. Test Alert Notifications

1. Buat alert dengan meningkatkan memory/disk usage
2. Harus muncul browser notification (jika allowed)
3. Harus muncul in-page notification
4. Harus play alert sound

### 4. Test Auto-Reconnect

1. Stop server: `docker-compose down`
2. Status harus berubah ke "Disconnected"
3. Start server: `docker-compose up -d`
4. Harus auto-reconnect dalam beberapa detik
5. Status kembali ke "Real-time Connected"

### 5. Test Fallback to Polling

1. Stop server
2. Tunggu 3 reconnect attempts (15-30 detik)
3. Status harus berubah ke "Using HTTP Polling"
4. Start server
5. Harus tetap polling (tidak auto-switch ke WebSocket)
6. Refresh page untuk kembali ke WebSocket mode

### 6. Test Multiple Tabs

1. Buka 2-3 tabs ke monitor page
2. Semua tabs harus connected
3. Update harus sinkron di semua tabs
4. Alert harus muncul di semua tabs

## Browser Compatibility

### WebSocket Support

- ✅ Chrome/Edge 16+
- ✅ Firefox 11+
- ✅ Safari 7+
- ✅ Opera 12.1+
- ✅ iOS Safari 7+
- ✅ Android Browser 4.4+

### Notification API

- ✅ Chrome/Edge 22+
- ✅ Firefox 22+
- ✅ Safari 6+
- ⚠️ iOS Safari - Tidak support (in-page notification tetap work)

### Audio API

- ✅ Chrome/Edge
- ✅ Firefox
- ✅ Safari
- ⚠️ Perlu user interaction untuk auto-play

## Troubleshooting

### WebSocket tidak connect

1. **Check server running**:

   ```bash
   docker-compose ps
   ```

2. **Check WebSocket endpoint**:

   ```bash
   curl -i -N -H "Connection: Upgrade" \
        -H "Upgrade: websocket" \
        -H "Sec-WebSocket-Version: 13" \
        -H "Sec-WebSocket-Key: test" \
        http://localhost:18904/ws
   ```

3. **Check browser console**:

   - Buka DevTools (F12)
   - Tab Console
   - Look for "WebSocket connected" atau error messages

4. **Check Network tab**:
   - DevTools → Network tab
   - Filter: WS (WebSocket)
   - Should see `/ws` connection with status 101 Switching Protocols

### Notifications tidak muncul

1. **Check browser permission**:

   ```javascript
   // Di console
   Notification.permission;
   // Harus: "granted"
   ```

2. **Request permission manually**:

   ```javascript
   Notification.requestPermission();
   ```

3. **Check notification container**:
   ```javascript
   document.getElementById("notifications");
   ```

### Alert sound tidak play

1. **User interaction required**: Click anywhere di page dulu
2. **Check browser console**: Ada error audio?
3. **Volume check**: Pastikan speaker tidak mute

### Reconnect terus menerus

1. **Check server logs**:

   ```bash
   docker-compose logs -f monserv
   ```

2. **Check firewall**: Port 18904 allowed?

3. **Check reverse proxy**: Pastikan WebSocket upgrade headers diforward

## Performance

### Bandwidth Usage

- **WebSocket**: ~1-2 KB per update (compressed JSON)
- **Polling**: ~3-5 KB per request (HTTP overhead)
- **Savings**: ~60% bandwidth reduction

### Latency

- **WebSocket**: <100ms dari server ke client
- **Polling**: 0-5 seconds delay (average 2.5s)
- **Improvement**: ~25x faster updates

### Connection Overhead

- **WebSocket**: 1 connection per client (persistent)
- **Polling**: 12 requests per minute per client
- **Server load**: ~90% reduction

## Best Practices

1. **Always handle reconnection**: Network issues are common
2. **Provide fallback**: Not all environments support WebSocket
3. **Show connection status**: User should know if data is real-time or delayed
4. **Graceful degradation**: App should work even if WebSocket fails
5. **Cleanup connections**: Close WebSocket on page unload
6. **Handle visibility**: Reconnect when tab becomes visible
7. **Rate limiting**: Don't spam reconnection attempts
8. **User notifications**: Inform user about connection issues

## Security Notes

1. **Use WSS in production**: Encrypted WebSocket (wss://)
2. **Validate origin**: Server should check Origin header
3. **Authentication**: Implement auth for WebSocket connections
4. **Rate limiting**: Prevent abuse with connection limits
5. **Input validation**: Validate all messages from server

## Next Steps

Fitur yang bisa ditambahkan:

1. **Authentication**: Login required untuk WebSocket
2. **Room/Channel**: Subscribe ke specific servers only
3. **History**: Buffer messages saat disconnected
4. **Compression**: Enable WebSocket compression
5. **Heartbeat**: Custom ping/pong for connection check
6. **Metrics**: Track WebSocket performance
7. **Error reporting**: Send errors ke monitoring system

## References

- [WebSocket API - MDN](https://developer.mozilla.org/en-US/docs/Web/API/WebSocket)
- [Notification API - MDN](https://developer.mozilla.org/en-US/docs/Web/API/Notifications_API)
- [Page Visibility API - MDN](https://developer.mozilla.org/en-US/docs/Web/API/Page_Visibility_API)
- [gorilla/websocket - GitHub](https://github.com/gorilla/websocket)
