const {test} = require('node:test');
const assert = require('node:assert/strict');
const vm = require('node:vm');
const fs = require('node:fs');

test('conference clock selects days and meals, while preserving manual selection', () => {
  let now = '2026-10-12T10:45:00Z'; // 12:45 in Berlin
  let tick;
  const button = (index, date) => ({dataset: {dayButton: String(index), date}, setAttribute() {}, querySelector: () => ({}), addEventListener(type, callback) {this.click = callback;}});
  const days = [button(0, '2026-10-12'), button(1, '2026-10-13')];
  const panels = days.map((day) => ({dataset: {dayPanel: day.dataset.dayButton}, classList: {toggle() {}}, meals: [
    {dataset: {featuredFrom: '08:00', featuredUntil: '10:00'}},
    {dataset: {featuredFrom: '12:30', featuredUntil: '14:00'}},
    {dataset: {featuredFrom: '18:00', featuredUntil: '20:00'}},
  ], querySelectorAll() {return this.meals;}}));
  const root = {lang: 'de'};
  const document = {documentElement: root, body: {dataset: {timezone: 'Europe/Berlin'}},
    querySelectorAll: (selector) => ({'[data-day-button]': days, '[data-day-panel]': panels}[selector] || []),
    querySelector: (selector) => [".app-version", ".brand"].includes(selector) ? null : ({setAttribute() {}}), getElementById: () => null, addEventListener() {}};
  class ClockDate extends Date { constructor(value) {super(value === undefined ? now : value);} }
  vm.runInNewContext(fs.readFileSync('web/static/app.js', 'utf8'), {document, Date: ClockDate, Intl, navigator: {languages: ['de']}, location: {hash: ''}, window: {addEventListener() {}}, setInterval: (callback) => {tick = callback;}});
  assert.equal(panels[0].hidden, false);
  assert.equal(panels[0].meals[1].hidden, false);
  now = '2026-10-12T12:00:00Z'; tick(); // lunch ends, dinner next
  assert.equal(panels[0].meals[2].hidden, false);
  now = '2026-10-12T22:01:00Z'; tick(); // midnight in Berlin
  assert.equal(panels[1].hidden, false);
  assert.equal(panels[1].meals[0].hidden, false);
  days[0].click(); tick();
  assert.equal(panels[0].hidden, false);
  assert.equal(panels[0].meals[2].hidden, false);
});
