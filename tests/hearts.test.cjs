const {test} = require('node:test');
const assert = require('node:assert/strict');
const vm = require('node:vm');
const fs = require('node:fs');

test('heart grows on each click and starts the shower only on the tenth', () => {
  const nodes = [];
  let duration;
  const document = {
    documentElement: {lang: 'en'}, body: {append() {}},
    createElement(tag) {
      const node = {tag, dataset: {}, setAttribute() {}, append() {}, showModal() {}, focus() {}, close() {}, remove() {this.removed = true;}, addEventListener(type, cb) {this[type] = cb;}, getContext: () => ({setTransform() {}})};
      nodes.push(node); return node;
    },
  };
  const context = vm.createContext({document, window: {innerWidth: 400, innerHeight: 800, matchMedia: () => ({matches: false}), addEventListener() {}, removeEventListener() {}}, performance: {now: () => 0}, requestAnimationFrame: () => 1, cancelAnimationFrame() {}, setTimeout: (_, ms) => {duration = ms;}, clearTimeout() {}});
  vm.runInContext(fs.readFileSync('web/static/hearts.js', 'utf8').replace('export function startHearts', 'function startHearts')+'\nstartHearts();', context);
  const heart = nodes.find(n => n.className === 'growing-heart');
  for (let i = 1; i <= 9; i++) {
    heart.click(); assert.equal(heart.dataset.clicks, String(i));
    assert.equal(nodes.filter(n => n.tag === 'canvas').length, 0);
  }
  heart.click();
  assert.equal(nodes.filter(n => n.tag === 'canvas').length, 1);
  assert.equal(nodes[0].removed, true);
  assert.equal(duration, 30000);
});

test('shower draws a fixed mix of vector hearts and sparkles without text', () => {
  let frame, hearts = 0, sparkles = 0, curves = 0, lines = 0;
  const colors = {};
  const painter = {
    setTransform() {}, clearRect() {}, save() {}, restore() {}, translate() {}, scale() {},
    beginPath() {curves = 0; lines = 0;}, moveTo() {},
    bezierCurveTo() {curves++;}, lineTo() {lines++;}, closePath() {}, stroke() {},
    fill() {if (curves) hearts++; if (lines) sparkles++; colors[this.fillStyle] = (colors[this.fillStyle] || 0) + 1;},
  };
  const context = vm.createContext({
    document: {createElement: () => ({setAttribute() {}, getContext: () => painter}), body: {append() {}}},
    window: {innerWidth: 400, innerHeight: 800, matchMedia: () => ({matches: false}), addEventListener() {}},
    performance: {now: () => 0}, setTimeout() {}, requestAnimationFrame: (cb) => {frame = cb;},
  });
  vm.runInContext(fs.readFileSync('web/static/hearts.js', 'utf8').replace('export function startHearts', 'function startHearts')+'\nstartShower();', context);
  frame(16);
  assert.equal(hearts, 40);
  assert.equal(sparkles, 40);
  assert.deepEqual(Object.values(colors), [20, 20, 20, 20]);
});
