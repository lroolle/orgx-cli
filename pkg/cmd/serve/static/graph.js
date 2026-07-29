// The local graph view, the way Logseq/Obsidian taught people to
// read one: force-directed, hover highlights the neighborhood,
// click opens the node. Vanilla canvas — no dependency, works
// offline, and a vault-sized graph does not need more.
(async () => {
  const canvas = document.getElementById("graph");
  if (!canvas) return;
  const res = await fetch("graph.json");
  const g = await res.json();
  const vault = canvas.dataset.vault;
  const css = getComputedStyle(document.body);
  const color = {
    node: css.getPropertyValue("--graph-node").trim() || "#3b6ea5",
    journal: css.getPropertyValue("--graph-journal").trim() || "#9aa2aa",
    edge: css.getPropertyValue("--graph-edge").trim() || "rgba(110,120,130,0.25)",
    label: css.getPropertyValue("--fg").trim() || "#24292e",
  };

  const isJournal = (n) =>
    /[\\/](journals|daily)[\\/]/.test(n.path) || /^\d{4}-\d{2}-\d{2}$/.test(n.title);

  const all = (g.nodes || []).map((n, i) => ({
    id: n.id,
    title: n.title || n.id,
    journal: isJournal(n),
    degree: 0,
    x: Math.cos((i / (g.nodes.length || 1)) * 2 * Math.PI) * 200,
    y: Math.sin((i / (g.nodes.length || 1)) * 2 * Math.PI) * 200,
    vx: 0,
    vy: 0,
  }));
  const byID = new Map(all.map((n) => [n.id, n]));
  for (const e of g.edges || []) {
    const a = byID.get(e.from), b = byID.get(e.to);
    if (a && b) { a.degree++; b.degree++; }
  }

  let nodes = all, edges = g.edges || [], alpha = 1;
  const toggle = document.getElementById("show-journals");
  const rebuild = () => {
    const show = !toggle || toggle.checked;
    nodes = all.filter((n) => show || !n.journal);
    const visible = new Set(nodes.map((n) => n.id));
    edges = (g.edges || []).filter((e) => visible.has(e.from) && visible.has(e.to));
    alpha = 1;
  };
  if (toggle) toggle.addEventListener("change", rebuild);
  rebuild();

  const dpr = window.devicePixelRatio || 1;
  const ctx = canvas.getContext("2d");
  let w, h;
  const resize = () => {
    w = canvas.clientWidth; h = canvas.clientHeight;
    canvas.width = w * dpr; canvas.height = h * dpr;
    ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
  };
  window.addEventListener("resize", () => { resize(); alpha = Math.max(alpha, 0.3); });
  resize();

  const radius = (n) => 3 + Math.min(6, Math.sqrt(n.degree) * 1.5);
  let hover = null;

  const step = () => {
    if (alpha > 0.003) {
      for (let i = 0; i < nodes.length; i++) {
        const a = nodes[i];
        for (let j = i + 1; j < nodes.length; j++) {
          const b = nodes[j];
          let dx = a.x - b.x, dy = a.y - b.y;
          let d2 = dx * dx + dy * dy || 1;
          if (d2 < 90000) {
            const f = (900 / d2) * alpha;
            dx *= f; dy *= f;
            a.vx += dx; a.vy += dy; b.vx -= dx; b.vy -= dy;
          }
        }
        a.vx -= a.x * 0.003 * alpha;
        a.vy -= a.y * 0.003 * alpha;
      }
      for (const e of edges) {
        const a = byID.get(e.from), b = byID.get(e.to);
        if (!a || !b) continue;
        const dx = b.x - a.x, dy = b.y - a.y;
        const d = Math.sqrt(dx * dx + dy * dy) || 1;
        const f = ((d - 60) / d) * 0.02 * alpha;
        a.vx += dx * f; a.vy += dy * f; b.vx -= dx * f; b.vy -= dy * f;
      }
      for (const n of nodes) {
        n.x += n.vx; n.y += n.vy; n.vx *= 0.85; n.vy *= 0.85;
      }
      alpha *= 0.985;
    }

    ctx.clearRect(0, 0, w, h);
    ctx.save();
    ctx.translate(w / 2, h / 2);

    const neighbors = new Set();
    if (hover) {
      neighbors.add(hover.id);
      for (const e of edges) {
        if (e.from === hover.id) neighbors.add(e.to);
        if (e.to === hover.id) neighbors.add(e.from);
      }
    }

    for (const e of edges) {
      const a = byID.get(e.from), b = byID.get(e.to);
      if (!a || !b) continue;
      const lit = hover && (e.from === hover.id || e.to === hover.id);
      ctx.strokeStyle = lit ? color.node : color.edge;
      ctx.lineWidth = lit ? 1.5 : 1;
      ctx.beginPath();
      ctx.moveTo(a.x, a.y);
      ctx.lineTo(b.x, b.y);
      ctx.stroke();
    }
    for (const n of nodes) {
      const faded = hover && !neighbors.has(n.id);
      ctx.globalAlpha = faded ? 0.25 : 1;
      ctx.fillStyle = n.journal ? color.journal : color.node;
      ctx.beginPath();
      ctx.arc(n.x, n.y, radius(n), 0, 2 * Math.PI);
      ctx.fill();
      ctx.globalAlpha = 1;
    }
    if (hover) {
      ctx.fillStyle = color.label;
      ctx.font = "13px system-ui, sans-serif";
      ctx.fillText(hover.title, hover.x + 9, hover.y + 4);
    }
    ctx.restore();
    requestAnimationFrame(step);
  };

  const pick = (ev) => {
    const rect = canvas.getBoundingClientRect();
    const x = ev.clientX - rect.left - w / 2;
    const y = ev.clientY - rect.top - h / 2;
    let best = null, bestD = 12 * 12;
    for (const n of nodes) {
      const dx = n.x - x, dy = n.y - y;
      const d = dx * dx + dy * dy;
      if (d < bestD) { best = n; bestD = d; }
    }
    return best;
  };
  canvas.addEventListener("mousemove", (ev) => {
    hover = pick(ev);
    canvas.style.cursor = hover ? "pointer" : "default";
  });
  canvas.addEventListener("mouseleave", () => { hover = null; });
  canvas.addEventListener("click", (ev) => {
    const n = pick(ev);
    if (n) window.location.href = "/v/" + encodeURIComponent(vault) + "/node/" + encodeURIComponent(n.id);
  });

  requestAnimationFrame(step);
})();
