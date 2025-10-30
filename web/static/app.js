import { fetchState } from './api.js';
import { render } from './render.js';

// WebSocket connection
let ws = null;
let reconnectTimer = null;
let useWebSocket = true; // Toggle WebSocket/Polling
let connectionAttempts = 0;

// Fallback polling function
async function loop() {
  try {
    const state = await fetchState();
    render(state);
    updateTimestamp();
    updateConnectionStatus('polling', 'Using HTTP Polling');
  } catch (e) {
    console.error('Polling error:', e);
    updateConnectionStatus('error', 'Connection Error');
  } finally {
    if (!useWebSocket) {
      setTimeout(loop, 5000);
    }
  }
}

// WebSocket connection function
function connectWebSocket() {
  if (ws && ws.readyState === WebSocket.OPEN) {
    return;
  }

  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  const wsUrl = `${protocol}//${window.location.host}/ws`;
  
  console.log('Connecting to WebSocket:', wsUrl);
  updateConnectionStatus('connecting', 'Connecting...');
  
  try {
    ws = new WebSocket(wsUrl);
    
    ws.onopen = () => {
      console.log('WebSocket connected');
      connectionAttempts = 0;
      updateConnectionStatus('connected', 'Real-time Connected');
      
      // Show notification
      showNotification('Connected', 'Real-time updates enabled', 'success');
    };
    
    ws.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        
        if (data.type === 'metrics_update') {
          // Render metrics update
          render(data.data);
          updateTimestamp();
        } else if (data.type === 'alert') {
          // Show alert notification
          handleAlert(data);
        }
      } catch (e) {
        console.error('Error parsing WebSocket message:', e);
      }
    };
    
    ws.onerror = (error) => {
      console.error('WebSocket error:', error);
      updateConnectionStatus('error', 'Connection Error');
    };
    
    ws.onclose = (event) => {
      console.log('WebSocket closed:', event.code, event.reason);
      updateConnectionStatus('disconnected', 'Disconnected');
      
      // Attempt to reconnect
      connectionAttempts++;
      const delay = Math.min(5000 * connectionAttempts, 30000); // Max 30s
      
      console.log(`Reconnecting in ${delay/1000}s... (attempt ${connectionAttempts})`);
      
      clearTimeout(reconnectTimer);
      reconnectTimer = setTimeout(() => {
        if (useWebSocket) {
          connectWebSocket();
        }
      }, delay);
      
      // Fallback to polling after 3 failed attempts
      if (connectionAttempts >= 3) {
        console.log('Falling back to HTTP polling...');
        useWebSocket = false;
        loop();
      }
    };
    
  } catch (e) {
    console.error('Failed to create WebSocket:', e);
    updateConnectionStatus('error', 'Failed to connect');
    
    // Fallback to polling
    useWebSocket = false;
    loop();
  }
}

// Disconnect WebSocket
function disconnectWebSocket() {
  if (ws) {
    ws.close();
    ws = null;
  }
  clearTimeout(reconnectTimer);
}

// Update timestamp display
function updateTimestamp() {
  const timestampEl = document.getElementById('updated');
  if (timestampEl) {
    timestampEl.textContent = 'Updated: ' + new Date().toLocaleString();
  }
}

// Update connection status indicator
function updateConnectionStatus(status, text) {
  const statusEl = document.getElementById('connection-status');
  const statusDot = document.getElementById('status-dot');
  const statusText = document.getElementById('status-text');
  
  if (statusDot) {
    statusDot.className = `status-dot ${status}`;
  }
  
  if (statusText) {
    statusText.textContent = text;
  }
  
  // Update badge if exists
  const badge = document.querySelector('.connection-badge');
  if (badge) {
    badge.className = `connection-badge ${status}`;
    badge.textContent = text;
  }
}

// Handle alert notifications
function handleAlert(alert) {
  const type = alert.alert_type === 'recovery' ? 'success' : 'warning';
  showNotification(alert.subject, alert.message, type);
  
  // Log to console
  console.log(`[${alert.alert_type.toUpperCase()}]`, alert.subject, '-', alert.message);
  
  // Play sound (optional)
  if (alert.alert_type === 'alert') {
    playAlertSound();
  }
}

// Show notification
function showNotification(title, message, type = 'info') {
  // Try to use browser notifications
  if ('Notification' in window && Notification.permission === 'granted') {
    new Notification(title, {
      body: message,
      icon: '/static/icon.png',
      tag: 'monserv-notification'
    });
  }
  
  // Show in-page notification
  const container = document.getElementById('notifications');
  if (container) {
    const notification = document.createElement('div');
    notification.className = `notification notification-${type}`;
    notification.innerHTML = `
      <strong>${title}</strong>
      <p>${message}</p>
    `;
    
    container.appendChild(notification);
    
    // Auto remove after 5 seconds
    setTimeout(() => {
      notification.remove();
    }, 5000);
  }
}

// Play alert sound
function playAlertSound() {
  try {
    const audio = new Audio('data:audio/wav;base64,UklGRnoGAABXQVZFZm10IBAAAAABAAEAQB8AAEAfAAABAAgAZGF0YQoGAACBhYqFbF1fdJivrJBhNjVgodDbq2EcBj+a2/LDciUFLIHO8tiJNwgZaLvt559NEAxQp+PwtmMcBjiR1/LMeSwFJHfH8N2QQAoUXrTp66hVFApGn+DyvmwhBTGH0fPTgjMGHm7A7+OZURE=');
    audio.volume = 0.3;
    audio.play().catch(e => console.log('Could not play alert sound:', e));
  } catch (e) {
    console.log('Alert sound not available');
  }
}

// Request notification permission
function requestNotificationPermission() {
  if ('Notification' in window && Notification.permission === 'default') {
    Notification.requestPermission();
  }
}

// Toggle between WebSocket and Polling
window.toggleConnectionMode = function() {
  useWebSocket = !useWebSocket;
  
  if (useWebSocket) {
    console.log('Switching to WebSocket mode...');
    disconnectWebSocket();
    connectionAttempts = 0;
    connectWebSocket();
  } else {
    console.log('Switching to Polling mode...');
    disconnectWebSocket();
    loop();
  }
};

// Initialize on page load
window.addEventListener('load', () => {
  // Load thresholds from data attributes
  const th = document.getElementById('th');
  if (th) {
    const m = parseFloat(th.dataset.mem || '90');
    const d = parseFloat(th.dataset.disk || '90');
    const p = parseFloat(th.dataset.proc || '20');
    if (!Number.isNaN(m)) window.MEM_TH = m;
    if (!Number.isNaN(d)) window.DISK_TH = d;
    if (!Number.isNaN(p)) window.PROC_TH = p;
  }
  
  // Request notification permission
  requestNotificationPermission();
  
  // Try WebSocket first, fallback to polling
  connectWebSocket();
  
  // Add visibility change handler to reconnect when tab becomes visible
  document.addEventListener('visibilitychange', () => {
    if (!document.hidden && useWebSocket) {
      if (!ws || ws.readyState !== WebSocket.OPEN) {
        console.log('Tab visible, reconnecting WebSocket...');
        connectWebSocket();
      }
    }
  });
});

// Cleanup on page unload
window.addEventListener('beforeunload', () => {
  disconnectWebSocket();
});
