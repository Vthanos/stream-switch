(() => {
    let ws = null;
    let paused = false;

    const $ = (id) => document.getElementById(id);

    const baseUrl = $("baseUrl");
    const sensor = $("sensor");
    const statusTxt = $("statusTxt");
    const statusDot = $("statusDot");

    const tpsVal = $("tpsVal");
    const tpsMax = $("tpsMax");
    const tpsAvg = $("tpsAvg");
    const p50El = $("p50");
    const p95El = $("p95");
    const jitterEl = $("jitter");

    const logEl = $("log");
    const themeCb = $("themeCb");

    const btnConnect = $("btnConnect");
    const btnDisconnect = $("btnDisconnect");
    const btnReset = $("btnReset");
    const btnGo = $("btnGo");
    const btnSpring = $("btnSpring");
    const btnPause = $("btnPause");
    const btnResume = $("btnResume");

    // THEME
    themeCb.addEventListener("change", () => {
        document.documentElement.setAttribute("data-theme", themeCb.checked ? "light" : "dark");
    });

    // CHART STATE
    const tpsCtx = $("tpsChart").getContext("2d");
    const latCtx = $("latChart").getContext("2d");
    const tpsSeries = [];
    const p95Series = [];
    const maxPoints = 80;
    let lastSec = Date.now();
    let msgCount = 0, maxTps = 0;
    let totalTpsTicks = 0, sumTps = 0;
    const latWindow = []; // last N ms for percentile
    const winSize = 2000; // rolling latency sample for percentiles

    function drawSeries(ctx, series, color="#7c5cff") {
        const w = ctx.canvas.width = ctx.canvas.clientWidth;
        const h = ctx.canvas.height = ctx.canvas.clientHeight;
        ctx.clearRect(0,0,w,h);
        if (!series.length) return;
        const maxV = Math.max(...series);
        const minV = Math.min(...series);
        const pad = 8;
        ctx.beginPath();
        for (let i=0;i<series.length;i++) {
            const x = pad + (w-2*pad) * (i/(series.length-1));
            const y = h - pad - (h-2*pad) * ((series[i]-minV)/Math.max(1e-9,(maxV-minV)));
            if (i===0) ctx.moveTo(x,y); else ctx.lineTo(x,y);
        }
        ctx.strokeStyle = color; ctx.lineWidth = 2; ctx.stroke();
    }

    function setStatus(on) {
        statusTxt.textContent = on ? "Connected" : "Disconnected";
        statusDot.className = "status-dot " + (on ? "dot-on" : "dot-off");
    }

    function percentile(arr, p) {
        if (!arr.length) return 0;
        const i = Math.floor(p * (arr.length - 1));
        return arr.slice().sort((a,b)=>a-b)[i];
    }

    function jitter(arr) {
        if (arr.length < 2) return 0;
        let j = 0;
        for (let i=1;i<arr.length;i++) j += Math.abs(arr[i]-arr[i-1]);
        return j/(arr.length-1);
    }

    function connect() {
        try { if (ws) ws.close(); } catch(_) {}
        const base = baseUrl.value.trim();
        const filt = sensor.value.trim() || "*";
        const url = `${base}/ws/subscribe?sensor_id=${encodeURIComponent(filt)}`;
        ws = new WebSocket(url);

        ws.onopen = () => setStatus(true);
        ws.onclose = () => setStatus(false);

        ws.onmessage = (ev) => {
            if (paused) return;
            const f = JSON.parse(ev.data);
            msgCount++;

            const serverSend = (f.meta?.sentUnixNano ?? f.meta?.receivedUnixNano ?? 0);
            const clientRecv = f.client_recv_unix_nano ?? (Date.now()*1e6);
            const ms = Math.max(0, clientRecv - serverSend) / 1e6;

            // Log line
            const line = `[${new Date().toLocaleTimeString()}] ${f.reading?.sensorId} #${f.reading?.seq} v=${Number(f.reading?.value).toFixed(2)} ms=${ms.toFixed(2)}`;
            const div = document.createElement("div");
            div.textContent = line;
            logEl.prepend(div);
            while (logEl.children.length > 150) logEl.removeChild(logEl.lastChild);

            // latency window
            latWindow.push(ms);
            if (latWindow.length > winSize) latWindow.splice(0, latWindow.length - winSize);

            // once per second: update KPI + charts
            const now = Date.now();
            if (now - lastSec >= 1000) {
                const tps = msgCount;
                tpsSeries.push(tps);
                if (tpsSeries.length > maxPoints) tpsSeries.shift();
                tpsVal.textContent = String(tps);
                maxTps = Math.max(maxTps, tps);
                tpsMax.textContent = String(maxTps);
                totalTpsTicks++; sumTps += tps;
                tpsAvg.textContent = (sumTps/totalTpsTicks).toFixed(1);
                msgCount = 0; lastSec = now;
                drawSeries(tpsCtx, tpsSeries, "#5b7bff");

                const p50 = percentile(latWindow, 0.50);
                const p95 = percentile(latWindow, 0.95);
                p50El.textContent = p50.toFixed(2);
                p95El.textContent = p95.toFixed(2);
                jitterEl.textContent = jitter(latWindow.slice(-30)).toFixed(2);

                p95Series.push(p95);
                if (p95Series.length > maxPoints) p95Series.shift();
                drawSeries(latCtx, p95Series, "#34d399");
            }
        }
    }

    function disconnect() {
        try { ws && ws.close(); } catch(_) {}
        setStatus(false);
    }

    function resetUI() {
        tpsSeries.length = 0; p95Series.length = 0; latWindow.length = 0;
        tpsVal.textContent = "0"; tpsMax.textContent = "0"; tpsAvg.textContent = "0";
        p50El.textContent = "-"; p95El.textContent = "-"; jitterEl.textContent = "-";
        logEl.innerHTML = "";
        drawSeries(tpsCtx, [], "#5b7bff");
        drawSeries(latCtx, [], "#34d399");
    }

    // Buttons
    btnConnect.addEventListener("click", connect);
    btnDisconnect.addEventListener("click", disconnect);
    btnReset.addEventListener("click", resetUI);
    btnGo.addEventListener("click", () => { baseUrl.value = "ws://localhost:8080"; });
    btnSpring.addEventListener("click", () => { baseUrl.value = "ws://localhost:8081"; });
    btnPause.addEventListener("click", () => { paused = true; });
    btnResume.addEventListener("click", () => { paused = false; });

    // Auto-connect once on load
    connect();
})();
