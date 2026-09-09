const {test} = require('node:test');
const assert = require('node:assert/strict');
const vm = require('node:vm');
const fs = require('node:fs');

test('theme is fetched only after a full hold and reused on subsequent toggles', () => {
  const handlers = {};
  let callback, delay, downloads = 0, link, shown = false, message;
  const source = fs.readFileSync('web/static/app.js', 'utf8').split('// A deliberate logo hold')[1];
  vm.runInNewContext('// A deliberate logo hold'+source, {
    document: {
      querySelector: (selector) => selector === '.brand' ? {addEventListener: (name, fn) => {handlers[name] = fn;}} : {after: (node) => {message = node;}},
      createElement: () => ({setAttribute() {}, remove() {}}),
      head: {append: (node) => {downloads++; link = node;}},
      documentElement: {classList: {add: () => {shown = true;}, toggle: () => {shown = !shown;}}},
    }, setTimeout: (fn, ms) => {callback = fn; delay = ms; return 1;}, clearTimeout: () => {callback = null;},
  });
  assert.equal(downloads, 0);
  handlers.pointerdown({button: 0, clientX: 0, clientY: 0});
  assert.equal(delay, 3000); assert.equal(downloads, 0);
  handlers.pointerup(); assert.equal(callback, null); assert.equal(downloads, 0);
  handlers.pointerdown({button: 0, clientX: 0, clientY: 0}); callback();
  assert.equal(downloads, 1); assert.equal(shown, false);
  link.onload(); assert.equal(shown, true); assert.equal(message.textContent, 'For the girly vibes');
  handlers.pointerdown({button: 0, clientX: 0, clientY: 0}); callback();
  assert.equal(shown, false); assert.equal(downloads, 1);
  assert.ok(!fs.readFileSync('web/templates/index.html', 'utf8').includes('girly.css'));
});
