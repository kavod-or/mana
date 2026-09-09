let stop = null;

function startShower() {
  if (stop) return stop;
    const canvas = document.createElement('canvas');
    canvas.className = 'manna-rain heart-shower';
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
    const flakes = Array.from({length: 80}, (_, index) => ({
      x: Math.random(), y: Math.random(), size: 14 + Math.random() * 16,
      heart: index % 2 === 0, color: ['#ff238e', '#d665ff', '#ffe38a', '#fff5fc'][Math.floor(index / 2) % 4],
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
        drawParticle(context, flake, flake.x * width + sway, flake.y * (height + 50) - 25);
      }
      frame = requestAnimationFrame(draw);
    }
    window.addEventListener('resize', resize);
    window.addEventListener('pagehide', finish);
    timer = setTimeout(finish, 30000);
    frame = requestAnimationFrame(draw);
    return finish;
}


// Normalized vector paths avoid platform-specific font and emoji rendering.
function drawParticle(context, flake, x, y) {
  context.save();
  context.translate(x, y);
  context.scale(flake.size, flake.size);
  context.beginPath();
  if (flake.heart) {
    context.moveTo(0, 0.45);
    context.bezierCurveTo(-0.12, 0.32, -0.5, 0.05, -0.5, -0.2);
    context.bezierCurveTo(-0.5, -0.52, -0.15, -0.58, 0, -0.3);
    context.bezierCurveTo(0.15, -0.58, 0.5, -0.52, 0.5, -0.2);
    context.bezierCurveTo(0.5, 0.05, 0.12, 0.32, 0, 0.45);
  } else {
    context.moveTo(0, -0.5);
    context.lineTo(0.12, -0.12);
    context.lineTo(0.5, 0);
    context.lineTo(0.12, 0.12);
    context.lineTo(0, 0.5);
    context.lineTo(-0.12, 0.12);
    context.lineTo(-0.5, 0);
    context.lineTo(-0.12, -0.12);
  }
  context.closePath();
  context.fillStyle = flake.color;
  context.strokeStyle = '#9d126a';
  context.lineWidth = 0.6 / flake.size;
  context.fill();
  context.stroke();
  context.restore();
}

export function startHearts() {
  if (stop) return () => { if (stop) stop(); };
  const dialog = document.createElement('dialog');
  dialog.className = 'heart-challenge';
  dialog.setAttribute('aria-label', 'For the girly vibes');
  const heart = document.createElement('button');
  heart.type = 'button';
  heart.className = 'growing-heart';
  const gem = document.createElement('span');
  gem.className = 'heart-gem';
  gem.textContent = '♥';
  gem.setAttribute('aria-hidden', 'true');
  heart.append(gem);
  heart.dataset.clicks = '0';
  const close = document.createElement('button');
  close.type = 'button';
  close.className = 'heart-close';
  close.textContent = '×';
  close.setAttribute('aria-label', document.documentElement.lang === 'de' ? 'Schließen' : 'Close');
  let clicks = 0;
  const update = () => {
    const label = document.documentElement.lang === 'de' ? 'Tippe auf das Herz' : 'Tap the heart';
    heart.setAttribute('aria-label', label);
    heart.dataset.clicks = String(clicks);
  };
  update();
  dialog.append(close, heart);
  document.body.append(dialog);
  const cleanup = () => {
    dialog.close();
    dialog.remove();
    window.removeEventListener('pagehide', finish);
    stop = null;
  };
  function finish() { if (stop) stop(); }
  stop = cleanup;
  close.addEventListener('click', cleanup);
  dialog.addEventListener('cancel', (event) => { event.preventDefault(); cleanup(); });
  heart.addEventListener('click', () => {
    clicks++;
    update();
    if (clicks === 10) { cleanup(); startShower(); }
  });
  window.addEventListener('pagehide', finish);
  dialog.showModal();
  heart.focus();
  return finish;
}
