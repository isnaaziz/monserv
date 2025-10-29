import { fetchState } from './api.js';
import { render } from './render.js';

async function loop() {
  try {
    const state = await fetchState();
    render(state);
    document.getElementById('updated').textContent = 'Updated: ' + new Date().toLocaleString();
  } catch (e) {
    console.error(e);
  } finally {
    setTimeout(loop, 5000);
  }
}

window.addEventListener('load', ()=>{
  const th = document.getElementById('th');
  if (th) {
    const m = parseFloat(th.dataset.mem || '90');
    const d = parseFloat(th.dataset.disk || '90');
    const p = parseFloat(th.dataset.proc || '20');
    if (!Number.isNaN(m)) window.MEM_TH = m;
    if (!Number.isNaN(d)) window.DISK_TH = d;
    if (!Number.isNaN(p)) window.PROC_TH = p;
  }
  loop();
});
