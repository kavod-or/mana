let activeStop = null;
const celebrationDelay = 2000;
const effectDuration = 30000;

export function startMazelTov(onStop = () => {}) {
  if (activeStop) return activeStop;

  const glass = document.createElement('div');
  glass.className = 'mazel-tov-glass';
  glass.setAttribute('aria-hidden', 'true');
  document.body.append(glass);

  let canvas;
  let message;
  let frame = 0;
  let revealTimer;
  let durationTimer;
  let stopped = false;
  let resize = () => {};

  const stop = () => {
    if (stopped) return;
    stopped = true;
    clearTimeout(revealTimer);
    clearTimeout(durationTimer);
    cancelAnimationFrame(frame);
    window.removeEventListener('resize', resize);
    window.removeEventListener('pagehide', stop);
    glass.remove();
    canvas?.remove();
    message?.remove();
    activeStop = null;
    onStop();
  };
  activeStop = stop;
  window.addEventListener('pagehide', stop);

  revealTimer = setTimeout(() => {
    if (stopped) return;
    canvas = document.createElement('canvas');
    canvas.className = 'mazel-tov-confetti';
    canvas.setAttribute('aria-hidden', 'true');
    message = document.createElement('div');
    message.className = 'mazel-tov-message';
    message.setAttribute('role', 'status');
    message.textContent = 'MAZEL TOV!';
    document.body.append(canvas, message);

    const context = canvas.getContext('2d');
    if (!context) return;
    const reducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches;
    let width;
    let height;
    resize = () => {
      width = window.innerWidth;
      height = window.innerHeight;
      const scale = Math.min(window.devicePixelRatio || 1, 2);
      canvas.width = width * scale;
      canvas.height = height * scale;
      context.setTransform(scale, 0, 0, scale, 0, 0);
    };
    resize();

    const colors = ['#1769aa', '#f5c542', '#e83e8c', '#32a852', '#ffffff'];
    const pieces = Array.from({length: 110}, (_, index) => ({
      x: Math.random(),
      y: reducedMotion ? Math.random() : -Math.random(),
      width: 6 + Math.random() * 8,
      height: 10 + Math.random() * 14,
      speed: 0.12 + Math.random() * 0.22,
      sway: Math.random() * Math.PI * 2,
      spin: (Math.random() - 0.5) * 5,
      color: colors[index % colors.length],
    }));
    let previous = performance.now();
    const draw = (now) => {
      const seconds = Math.min((now - previous) / 1000, 0.1);
      previous = now;
      context.clearRect(0, 0, width, height);
      for (const piece of pieces) {
        if (!reducedMotion) piece.y = (piece.y + seconds * piece.speed) % 1.15;
        const x = piece.x * width + (reducedMotion ? 0 : Math.sin(now / 700 + piece.sway) * 24);
        const y = piece.y * (height + 80) - 40;
        context.save();
        context.translate(x, y);
        context.rotate(reducedMotion ? 0 : now / 1000 * piece.spin);
        context.fillStyle = piece.color;
        context.fillRect(-piece.width / 2, -piece.height / 2, piece.width, piece.height);
        context.restore();
      }
      if (!reducedMotion) frame = requestAnimationFrame(draw);
    };
    window.addEventListener('resize', resize);
    frame = requestAnimationFrame(draw);
  }, celebrationDelay);
  durationTimer = setTimeout(stop, effectDuration);

  return stop;
}
