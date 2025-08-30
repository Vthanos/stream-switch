(function(){
  let ws = null;
  const statusEl = document.getElementById('status');
  const tpsEl = document.getElementById('tps');
  const p50El = document.getElementById('p50');
  const p95El = document.getElementById('p95');
  const logEl = document.getElementById('log');
  const backendSel = document.getElementById('backend');
  const sensorInput = document.getElementById('sensor');
  document.getElementById('connectBtn').onclick = connect;

  let lastSec = Date.now(), msgCount = 0;
  let latencies = []; // ms
  const maxPoints = 60;
  const tpsSeries = [];
  const latSeries = [];

  const tpsCtx = document.getElementById('tpsChart').getContext('2d');
  const latCtx = document.getElementById('latChart').getContext('2d');

  function drawSeries(ctx, series) {
    const w = ctx.canvas.width = ctx.canvas.clientWidth;
    const h = ctx.canvas.height = ctx.canvas.clientHeight;
    ctx.clearRect(0,0,w,h);
    const n = series.length;
    if (n === 0) return;
    const maxV = Math.max(...series);
    const minV = Math.min(...series);
    const pad = 8;
    ctx.beginPath();
    for (let i=0;i<n;i++) {
      const x = pad + (w-2*pad) * (i/(n-1));
      const y = h - pad - (h-2*pad) * ((series[i]-minV)/Math.max(1e-9,(maxV-minV)));
      if (i===0) ctx.moveTo(x,y); else ctx.lineTo(x,y);
    }
    ctx.strokeStyle = "#7dd3fc";
    ctx.lineWidth = 2;
    ctx.stroke();
  }

  function connect(){
    if (ws) { try { ws.close(); } catch(_){} }
    const base = backendSel.value;
    const sensor = sensorInput.value || "*";
    ws = new WebSocket(`${base}/ws/subscribe?sensor_id=${encodeURIComponent(sensor)}`);
    ws.onopen = () => { statusEl.textContent = "connected"; statusEl.className = "pill"; };
    ws.onclose = () => { statusEl.textContent = "disconnected"; statusEl.className = "pill"; };
    ws.onmessage = (ev) => {
      const f = JSON.parse(ev.data);
      msgCount++;
      const serverSend = f.meta?.sentUnixNano || f.meta?.receivedUnixNano || 0;
      const clientRecv = f.client_recv_unix_nano || (Date.now()*1e6);
      const latMs = Math.max(0, clientRecv - serverSend) / 1e6;
      latencies.push(latMs);
      if (latencies.length > 5000) latencies.splice(0, latencies.length-5000);

      const line = `[${new Date().toLocaleTimeString()}] id=${f.reading.sensorId} seq=${f.reading.seq} val=${f.reading.value.toFixed(2)} ms=${latMs.toFixed(2)}`;
      const div = document.createElement('div'); div.textContent = line;
      logEl.prepend(div);
      while (logEl.children.length > 100) logEl.removeChild(logEl.lastChild);

      const now = Date.now();
      if (now - lastSec >= 1000) {
        tpsSeries.push(msgCount);
        if (tpsSeries.length > maxPoints) tpsSeries.shift();
        tpsEl.textContent = String(msgCount);
        msgCount = 0; lastSec = now;
        drawSeries(tpsCtx, tpsSeries);

        const sample = latencies.slice(-1000).sort((a,b)=>a-b);
        function pct(p){ if(sample.length===0) return 0; const i=Math.floor(p*(sample.length-1)); return sample[i]; }
        const p50 = pct(0.50), p95 = pct(0.95);
        p50El.textContent = p50.toFixed(2);
        p95El.textContent = p95.toFixed(2);
        latSeries.push(p95);
        if (latSeries.length > maxPoints) latSeries.shift();
        drawSeries(latCtx, latSeries);
      }
    }
  }

  connect();
})();