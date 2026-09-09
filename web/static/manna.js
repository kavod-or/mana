let stop = null;

export function startManna() {
  if (stop) return;
    const canvas = document.createElement('canvas');
    canvas.className = 'manna-rain';
    canvas.setAttribute('aria-hidden', 'true');
    const context = canvas.getContext('2d');
    if (!context) return;
    document.body.append(canvas);

    const reducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)');
    let width, height;
    function resize() {
      width = window.innerWidth;
      height = window.innerHeight;
      const scale = Math.min(window.devicePixelRatio || 1, 2);
      canvas.width = width * scale;
      canvas.height = height * scale;
      context.setTransform(scale, 0, 0, scale, 0, 0);
    }
    resize();
    const flakes = Array.from({length: 65}, () => ({
      x: Math.random(), y: Math.random(), size: 3 + Math.random() * 6,
      speed: 0.07 + Math.random() * 0.12, phase: Math.random() * Math.PI * 2,
    }));
    let frame = 0;
    const started = performance.now();
    let previous = started;
    let timer;
    stop = () => {
      cancelAnimationFrame(frame);
      clearTimeout(timer);
      window.removeEventListener('resize', resize);
      window.removeEventListener('pagehide', finish);
      canvas.remove();
      stop = null;
    };
    function finish() { if (stop) stop(); }
    function draw(now) {
      if (now - started >= 30000) { finish(); return; }
      const seconds = Math.min((now - previous) / 1000, 0.1);
      previous = now;
      context.clearRect(0, 0, width, height);
      for (const flake of flakes) {
        if (!reducedMotion.matches) flake.y = (flake.y + seconds * flake.speed) % 1;
        const sway = reducedMotion.matches ? 0 : Math.sin(now / 1100 + flake.phase) * 18;
        context.beginPath();
        context.ellipse(flake.x * width + sway, flake.y * (height + 30) - 15, flake.size, flake.size * 0.65, flake.phase, 0, Math.PI * 2);
        context.fillStyle = '#e6bf78';
        context.fill();
        context.strokeStyle = '#c76443';
        context.lineWidth = 1;
        context.stroke();
      }
      frame = requestAnimationFrame(draw);
    }
    window.addEventListener('resize', resize);
    window.addEventListener('pagehide', finish);
    timer = setTimeout(finish, 30000);
    frame = requestAnimationFrame(draw);
}
