import { fmtBytes, badge } from './utils.js?v=2';
import { charts, historyData, updateHistory, createOrUpdateChart, createOrUpdateGauge } from './charts.js?v=2';

// Mask password in URL for safe display
function maskPassword(url) {
  try {
    const u = new URL(url);
    if (u.password) {
      u.password = '***';
    }
    return u.toString();
  } catch (e) {
    // Manual masking for URLs that don't parse
    if (url.includes('@') && url.includes(':')) {
      const parts = url.split('@');
      if (parts.length >= 2) {
        const authPart = parts[0];
        const restPart = parts.slice(1).join('@');
        const lastColon = authPart.lastIndexOf(':');
        if (lastColon > 0) {
          const beforePassword = authPart.substring(0, lastColon);
          return beforePassword + ':***@' + restPart;
        }
      }
    }
    return url;
  }
}

export function render(state) {
  const container = document.getElementById('cards');
  const agents = Object.keys(state).sort();
  const existingCardIds = new Set(Array.from(container.children).map(c => c.id));

  agents.forEach((url) => {
    const m = state[url];
    if (!m) return;
    console.log('Rendering agent:', url, 'CPU data:', m.cpu);
    const cardId = 'card-' + url.replace(/[^a-zA-Z0-9]/g, '-');
    const chartId = 'chart-' + url.replace(/[^a-zA-Z0-9]/g, '-');
    existingCardIds.delete(cardId);
    let card = document.getElementById(cardId);
    if (!card) {
      card = createAgentCardShell(url, m, cardId, chartId);
      container.appendChild(card);
    }
    updateAgentCardData(card, m, url, chartId);
    const cpuPct = (m.cpu || {}).usedPercent || 0;
    const memPct = (m.memory || {}).usedPercent || 0;
    const disks = m.disks || [];
    const avgDiskPct = disks.length > 0 ? disks.reduce((sum, d) => sum + (d.usedPercent || 0), 0) / disks.length : 0;
    updateHistory(url, cpuPct, memPct, avgDiskPct);
    createOrUpdateChart(chartId, url);
    createOrUpdateGauge(chartId + '-gauge', cpuPct, memPct, avgDiskPct);
  });
  existingCardIds.forEach(cardId => {
    const card = document.getElementById(cardId);
    if (card) card.remove();
    const chartId = cardId.replace('card-', 'chart-');
    if (charts[chartId]) {
      charts[chartId].destroy();
      delete charts[chartId];
    }
    if (charts[chartId + '-gauge']) {
      charts[chartId + '-gauge'].destroy();
      delete charts[chartId + '-gauge'];
    }
  });
}

export function createAgentCardShell(url, m, cardId, chartId) {
  const maskedUrl = maskPassword(url);
  const card = document.createElement('div');
  card.id = cardId;
  card.className = 'card';
  card.innerHTML = `
    <h2 class="agent-hostname">${m.hostname || url}</h2>
    <div class="small agent-meta">Agent: ${maskedUrl} • Uptime: ${m.uptimeSeconds||0}s • At: ${m.generatedAtUtc||''}</div>
    <div class="charts-row">
      <div class="chart-container">
        <div class="chart-title">CPU, Memory & Disk Usage Trend</div>
        <canvas id="${chartId}"></canvas>
      </div>
      <div class="chart-container">
        <div class="chart-title">Current Status</div>
        <canvas id="${chartId}-gauge"></canvas>
      </div>
    </div>
    <h3>Processor</h3>
    <div class="agent-cpu">Cores: - • Model: - • Usage: -</div>
    <h3>Memory</h3>
    <div class="agent-memory">Total: ${fmtBytes(0)} • Used: ${fmtBytes(0)} • Used: -</div>
    <h3>Disks</h3>
    <table class="agent-disks">
      <thead><tr><th>Mount</th><th>FS</th><th>Total</th><th>Used</th><th>Free</th><th>Used%</th></tr></thead>
      <tbody></tbody>
    </table>
    <h3>Top Procs by RAM</h3>
    <table class="agent-procs">
      <thead><tr><th>PID</th><th>Name</th><th>User</th><th>RSS</th><th>RAM%</th></tr></thead>
      <tbody></tbody>
    </table>
  `;
  return card;
}

export function updateAgentCardData(card, m, url, chartId) {
  const maskedUrl = maskPassword(url);
  const cpu = m.cpu || {};
  const mem = m.memory || {};
  const disks = m.disks || [];
  const procs = m.topProcsByMem || [];
  card.querySelector('.agent-hostname').textContent = m.hostname || url;
  card.querySelector('.agent-meta').innerHTML = `Agent: ${maskedUrl} • Uptime: ${m.uptimeSeconds||0}s • At: ${m.generatedAtUtc||''}`;
  card.querySelector('.agent-cpu').innerHTML = `Cores: ${cpu.cores||'-'} • Model: ${cpu.modelName||'-'} • Usage: ${cpu.usedPercent?badge(cpu.usedPercent, window.CPU_TH || 80):'-'}`;
  card.querySelector('.agent-memory').innerHTML = `Total: ${fmtBytes(mem.total||0)} • Used: ${fmtBytes(mem.used||0)} • Used: ${mem.usedPercent?badge(mem.usedPercent, window.MEM_TH || 90):'-'}`;
  const diskTbody = card.querySelector('.agent-disks tbody');
  diskTbody.innerHTML = disks.map(d=>`<tr><td>${d.mountpoint}</td><td>${d.fstype}</td><td>${fmtBytes(d.total)}</td><td>${fmtBytes(d.used)}</td><td>${fmtBytes(d.free)}</td><td>${badge(d.usedPercent, window.DISK_TH || 90)}</td></tr>`).join('');
  const procTbody = card.querySelector('.agent-procs tbody');
  procTbody.innerHTML = procs.map(p=>`<tr><td>${p.pid}</td><td>${p.name}</td><td>${p.username}</td><td>${fmtBytes(p.rssBytes)}</td><td>${badge(p.percentRAM||0, window.PROC_TH || 20)}</td></tr>`).join('');
}
