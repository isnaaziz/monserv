# Quick Start - Testing WebSocket Integration

## Prerequisites

- Docker & Docker Compose installed
- Port 18904 available
- Modern browser (Chrome, Firefox, Safari, Edge)

## Step 1: Start Services

```bash
cd /Users/macbookairm3/Documents/Isna\ Azis\ Nurohman/monserv
docker-compose up -d
```

## Step 2: Check Services Running

```bash
docker-compose ps
```

Expected output:

```
NAME       SERVICE    STATUS    PORTS
monserv    monserv    running   0.0.0.0:18904->18904/tcp
```

## Step 3: Open Web Monitor

1. Open browser: `http://localhost:18904`
2. Look at top-right corner
3. You should see: **🟢 Real-time Connected**

## Step 4: Test Real-time Updates

### Option A: Wait for Auto-Update (30 seconds)

Just wait and watch - metrics will update automatically without page refresh.

### Option B: Trigger Manual Update

```bash
# Check current metrics
docker-compose exec monserv ./monserv --help

# Or check logs
docker-compose logs -f monserv
```

## Step 5: Test WebSocket Connection

Open browser DevTools (F12) → Console and check:

```javascript
// Should see these messages:
"Connecting to WebSocket: ws://localhost:18904/ws";
"WebSocket connected";
```

Also check Network tab (filter: WS) - should see `/ws` with status 101.

## Step 6: Test Alert Notifications

### Enable Browser Notifications

1. Browser will ask for permission
2. Click "Allow" when prompted
3. Or manually enable in browser settings

### Trigger an Alert

To test alerts, you need a server with high resource usage:

```bash
# In your monitored server, run:
# For high memory:
stress --vm 2 --vm-bytes 1G --timeout 60s

# For high disk:
dd if=/dev/zero of=/tmp/test.img bs=1M count=10000

# For high CPU:
stress --cpu 4 --timeout 60s
```

When threshold is exceeded, you should see:

- 🔔 Browser notification
- In-page notification (top-right)
- Alert sound (beep)

## Step 7: Test Auto-Reconnect

### Simulate Connection Loss

```bash
# Stop server
docker-compose down
```

Browser should show:

- **⚫ Disconnected**
- Console: "Reconnecting in 5s... (attempt 1)"

### Restore Connection

```bash
# Start server
docker-compose up -d
```

Browser should automatically reconnect:

- **🟢 Real-time Connected**
- Console: "WebSocket connected"

## Step 8: Test Fallback to Polling

### Trigger Fallback

```bash
# Stop server
docker-compose down

# Wait 30-45 seconds (3 reconnect attempts)
```

Browser should switch to polling:

- **🔵 Using HTTP Polling**
- Updates continue but every 5 seconds

### Return to WebSocket

```bash
# Start server
docker-compose up -d

# Refresh browser page (F5)
```

Should reconnect with WebSocket:

- **🟢 Real-time Connected**

## Step 9: Test Multiple Clients

1. Open 3 browser tabs: `http://localhost:18904`
2. All tabs should show: **🟢 Real-time Connected**
3. Wait for an update
4. All tabs should update simultaneously

## Step 10: Monitor WebSocket Traffic

### Browser DevTools

1. F12 → Network tab
2. Filter: WS (WebSocket)
3. Click on `/ws` connection
4. See messages in "Messages" tab

### Server Logs

```bash
# Watch server logs
docker-compose logs -f monserv

# Should see:
# "WebSocket client connected"
# "Broadcasting metrics to N clients"
```

## Verification Checklist

- [ ] Services running (docker-compose ps)
- [ ] Web page loads (http://localhost:18904)
- [ ] Status shows "Real-time Connected" with green dot
- [ ] Metrics update automatically (no page refresh)
- [ ] Console shows "WebSocket connected"
- [ ] Network tab shows /ws with status 101
- [ ] Auto-reconnect works after disconnect
- [ ] Fallback to polling works after 3 failed attempts
- [ ] Multiple tabs work simultaneously
- [ ] Browser notifications work (if enabled)
- [ ] In-page notifications appear
- [ ] Alert sound plays

## Troubleshooting

### Status shows "Disconnected"

```bash
# Check if service running
docker-compose ps

# Check service logs
docker-compose logs monserv

# Restart service
docker-compose restart monserv
```

### Status shows "Connection Error"

```bash
# Check port not used by other app
lsof -i :18904

# Check firewall
sudo ufw status

# Check Docker network
docker network ls
docker network inspect pdk_service
```

### No notifications appearing

1. Check browser permission: DevTools → Application → Permissions
2. Enable notifications manually in browser settings
3. Check `#notifications` element exists in HTML

### Console shows errors

```bash
# Common errors:

# "WebSocket is closed before the connection is established"
→ Server not ready, wait and retry

# "CORS policy"
→ Check server CORS settings in main.go

# "Unexpected server response: 404"
→ Check /ws endpoint exists (grep for "/ws" in main.go)

# "Unexpected server response: 400"
→ Check WebSocket upgrade headers
```

## Performance Check

### Browser Performance

```javascript
// In console
performance.getEntriesByType("resource").filter((r) => r.name.includes("/ws"));
```

### Network Usage

```bash
# Check bandwidth
docker stats monserv

# Check connections
docker-compose exec monserv netstat -an | grep 18904
```

### WebSocket Stats

```javascript
// In console - custom function
function wsStats() {
  return {
    readyState: ws.readyState,
    bufferedAmount: ws.bufferedAmount,
    protocol: ws.protocol,
    url: ws.url,
  };
}
wsStats();
```

## Next Steps

Once everything works:

1. ✅ WebSocket real-time updates working
2. ✅ Auto-reconnect working
3. ✅ Fallback to polling working
4. ✅ Notifications working

You can:

- Deploy to production with HTTPS/WSS
- Add authentication to WebSocket
- Customize notification behavior
- Add more real-time features

## Quick Commands

```bash
# Start
docker-compose up -d

# Stop
docker-compose down

# Restart
docker-compose restart monserv

# Logs
docker-compose logs -f monserv

# Rebuild
docker-compose up -d --build

# Check status
curl http://localhost:18904/health

# Check metrics
curl http://localhost:18904/api/state

# Test WebSocket (using websocat)
websocat ws://localhost:18904/ws
```

## Success Indicators

You'll know everything works when:

1. **Green Status**: Top-right shows 🟢 "Real-time Connected"
2. **Live Updates**: Metrics update without refreshing
3. **Fast Response**: Updates appear within 1 second
4. **Auto Recovery**: Reconnects after server restart
5. **Multiple Clients**: Works with many tabs open
6. **Notifications**: Alerts appear as notifications
7. **Smooth UX**: No lag or freezing

## Support

If you encounter issues:

1. Check `docs/FRONTEND_WEBSOCKET.md` for detailed documentation
2. Check `docs/WEBSOCKET_API.md` for API reference
3. Check server logs: `docker-compose logs -f monserv`
4. Check browser console for errors
5. Check Network tab for WebSocket status

Happy monitoring! 🚀
