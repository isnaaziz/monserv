// Utility functions
export function fmtBytes(n) {
  if (n==null || isNaN(n)) return '-';
  const u = ['B','KB','MB','GB','TB'];
  let i=0, x=Number(n);
  while (x>=1024 && i<u.length-1) { x/=1024; i++; }
  return x.toFixed(1)+' '+u[i];
}

export function badge(pct, th) {
  const cls = pct>=th ? 'alert' : pct>=th-5 ? 'warn' : 'ok';
  return `<span class="${cls}">${pct.toFixed(1)}%</span>`;
}
