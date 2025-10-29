export const charts = {};
export const historyData = {};
export const MAX_HISTORY = 20;

export function updateHistory(url, mem, disk) {
  if (!historyData[url]) {
    historyData[url] = {
      time: [],
      mem: [],
      disk: []
    };
  }
  const h = historyData[url];
  const now = new Date().toLocaleTimeString();
  h.time.push(now);
  h.mem.push(mem);
  h.disk.push(disk);
  if (h.time.length > MAX_HISTORY) {
    h.time.shift();
    h.mem.shift();
    h.disk.shift();
  }
}

export function createOrUpdateChart(chartId, url) {
  const canvas = document.getElementById(chartId);
  if (!canvas) return;
  const h = historyData[url];
  if (!h || h.time.length === 0) return;
  const memTh = window.MEM_TH || 90;
  const diskTh = window.DISK_TH || 90;
  if (charts[chartId]) {
    charts[chartId].data.labels = h.time;
    charts[chartId].data.datasets[0].data = h.mem;
    charts[chartId].data.datasets[1].data = h.disk;
    charts[chartId].update('none');
  } else {
    const ctx = canvas.getContext('2d');
    charts[chartId] = new Chart(ctx, {
      type: 'line',
      data: {
        labels: h.time,
        datasets: [
          {
            label: 'Memory %',
            data: h.mem,
            borderColor: 'rgb(59, 130, 246)',
            backgroundColor: 'rgba(59, 130, 246, 0.1)',
            tension: 0.3,
            fill: true
          },
          {
            label: 'Disk %',
            data: h.disk,
            borderColor: 'rgb(16, 185, 129)',
            backgroundColor: 'rgba(16, 185, 129, 0.1)',
            tension: 0.3,
            fill: true
          }
        ]
      },
      options: {
        responsive: true,
        maintainAspectRatio: true,
        interaction: {
          intersect: false,
          mode: 'index'
        },
        plugins: {
          legend: {
            display: true,
            position: 'bottom',
            labels: { boxWidth: 12, font: { size: 11 } }
          },
          tooltip: {
            callbacks: {
              label: function(context) {
                return context.dataset.label + ': ' + context.parsed.y.toFixed(1) + '%';
              }
            }
          }
        },
        scales: {
          y: {
            beginAtZero: true,
            max: 100,
            ticks: {
              callback: function(value) { return value + '%'; },
              font: { size: 10 }
            },
            grid: { color: 'rgba(0,0,0,0.05)' }
          },
          x: {
            ticks: { 
              maxRotation: 45,
              minRotation: 45,
              font: { size: 9 }
            },
            grid: { display: false }
          }
        }
      }
    });
  }
}

export function createOrUpdateGauge(chartId, memPct, diskPct) {
  const canvas = document.getElementById(chartId);
  if (!canvas) return;
  const memTh = window.MEM_TH || 90;
  const diskTh = window.DISK_TH || 90;
  const memColor = memPct >= memTh ? 'rgb(220, 38, 38)' : (memPct >= memTh * 0.8 ? 'rgb(245, 158, 11)' : 'rgb(34, 197, 94)');
  const diskColor = diskPct >= diskTh ? 'rgb(220, 38, 38)' : (diskPct >= diskTh * 0.8 ? 'rgb(245, 158, 11)' : 'rgb(34, 197, 94)');
  if (charts[chartId]) {
    charts[chartId].data.datasets[0].data = [memPct, 100 - memPct];
    charts[chartId].data.datasets[0].backgroundColor[0] = memColor;
    charts[chartId].data.datasets[1].data = [diskPct, 100 - diskPct];
    charts[chartId].data.datasets[1].backgroundColor[0] = diskColor;
    charts[chartId].update('none');
  } else {
    const ctx = canvas.getContext('2d');
    charts[chartId] = new Chart(ctx, {
      type: 'doughnut',
      data: {
        labels: ['Memory', 'Disk'],
        datasets: [
          {
            label: 'Memory',
            data: [memPct, 100 - memPct],
            backgroundColor: [memColor, 'rgba(229, 231, 235, 0.3)'],
            borderWidth: 0,
            circumference: 180,
            rotation: 270
          },
          {
            label: 'Disk',
            data: [diskPct, 100 - diskPct],
            backgroundColor: [diskColor, 'rgba(229, 231, 235, 0.3)'],
            borderWidth: 0,
            circumference: 180,
            rotation: 270
          }
        ]
      },
      options: {
        responsive: true,
        maintainAspectRatio: true,
        cutout: '70%',
        plugins: {
          legend: { display: false },
          tooltip: {
            callbacks: {
              label: function(context) {
                if (context.dataIndex === 0) {
                  return context.dataset.label + ': ' + context.parsed.toFixed(1) + '%';
                }
                return null;
              }
            }
          }
        }
      },
      plugins: [{
        id: 'gaugeText',
        afterDraw: function(chart) {
          const ctx = chart.ctx;
          const centerX = (chart.chartArea.left + chart.chartArea.right) / 2;
          const centerY = (chart.chartArea.top + chart.chartArea.bottom) / 2 + 20;
          ctx.save();
          ctx.font = 'bold 11px sans-serif';
          ctx.fillStyle = '#111827';
          ctx.textAlign = 'center';
          ctx.textBaseline = 'middle';
          ctx.fillText('Mem: ' + memPct.toFixed(1) + '%', centerX, centerY - 10);
          ctx.fillText('Disk: ' + diskPct.toFixed(1) + '%', centerX, centerY + 10);
          ctx.restore();
        }
      }]
    });
  }
}
