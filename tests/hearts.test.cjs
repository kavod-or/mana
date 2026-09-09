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
