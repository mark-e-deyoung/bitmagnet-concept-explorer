import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import vm from 'node:vm';

const html = readFileSync(new URL('./index.html', import.meta.url), 'utf8');
const start = html.indexOf('function clearQueryEvidence(');
const end = html.indexOf('\nasync function capabilities()', start);
assert.ok(start >= 0 && end > start, 'production query-evidence clearer is present');
const helper = html.slice(start, end);
const outputs = [
  'status', 'direct', 'concept', 'overlap', 'external', 'details',
  'observed-graph', 'observed-list', 'observed-items', 'observed-status',
  'native-facet-data', 'native-facet-status', 'sweep-details', 'sweep-status',
];

for (const scenario of ['new request', 'timeout', 'HTTP 422', 'HTTP 401']) {
  const elements = new Map(outputs.map(id => [id, {
    textContent: 'stale evidence from previous query',
    children: ['stale child'],
    replaceChildren() { this.children = []; },
  }]));
  const context = {
    observedToken: 0,
    $: id => elements.get(id),
  };
  vm.runInNewContext(`let observedToken = 0; ${helper}; clearQueryEvidence(${JSON.stringify(scenario)}); observedToken`, context);
  for (const id of outputs.filter(id => id !== 'status')) {
    const element = elements.get(id);
    assert.equal(element.textContent, scenario, `${scenario} should replace stale text in ${id}`);
    assert.deepEqual(Array.from(element.children), [], `${scenario} should remove stale nodes from ${id}`);
  }
}

console.log('PASS: new request, timeout, HTTP 422, and HTTP 401 clear prior query evidence');
