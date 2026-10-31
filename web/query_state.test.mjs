import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import vm from 'node:vm';

const html = readFileSync(new URL('./index.html', import.meta.url), 'utf8');
const extract = (startMarker, endMarker) => {
  const start = html.indexOf(startMarker);
  const end = html.indexOf(endMarker, start);
  assert.ok(start >= 0 && end > start, `production source exists: ${startMarker}`);
  return html.slice(start, end);
};
const clearHelper = extract('function clearQueryEvidence(', '\nasync function capabilities()');
const clearIDs = [
  'status', 'direct', 'concept', 'overlap', 'external', 'details',
  'observed-graph', 'observed-list', 'observed-items', 'observed-status',
  'native-facet-data', 'native-facet-status', 'sweep-details', 'sweep-status',
];
const makeElements = () => new Map([...clearIDs, 'minseeders', 'minleechers', 'adult', 'sweep-run', 'sweep-plan', 'sweep-limits', 'languages', 'media', 'query'].map(id => [id, {
  textContent: 'stale evidence from previous query', children: [], value: '', disabled: false,
  listeners: {}, replaceChildren() { this.children = []; },
  addEventListener(name, listener) { this.listeners[name] = listener; },
}]));

for (const scenario of ['new request', 'timeout', 'HTTP 422', 'HTTP 401']) {
  const elements = makeElements();
  const context = { $: id => elements.get(id) };
  const generations = vm.runInNewContext(`let observedToken=0,facetToken=0,sweepToken=0,limitsToken=0; ${clearHelper}; clearQueryEvidence(${JSON.stringify(scenario)}); [observedToken,facetToken,sweepToken,limitsToken]`, context);
  assert.deepEqual(Array.from(generations), [1, 1, 1, 1]);
  for (const id of clearIDs.filter(id => id !== 'status')) {
    assert.equal(elements.get(id).textContent, scenario, `${scenario} clears ${id}`);
    assert.deepEqual(elements.get(id).children, [], `${scenario} removes nodes from ${id}`);
  }
}

function deferred() {
  let resolve;
  const promise = new Promise(r => { resolve = r; });
  return { promise, resolve };
}

async function proveLateResponseCannotRestore(widget, start, end, setup) {
  const fn = extract(start, end);
  const elements = makeElements();
  const pending = deferred();
  const context = {
    $: id => elements.get(id),
    fetch: () => pending.promise,
    AbortController,
    setTimeout,
    clearTimeout,
    buildSweepRequest: () => ({ query: 'synthetic', include_searxng: false, external_consent: false }),
  };
  const invoke = vm.runInNewContext(`${setup}
    let observedToken=0,facetToken=0,sweepToken=0,limitsToken=0,canExecute=true;
    ${clearHelper}
    ${fn}
    ${widget === 'native facets' ? 'loadNativeFacets()' : 'sweepAction(false)'}`, context);
  elements.get('minseeders').value = '';
  elements.get('minleechers').value = '';
  elements.get('adult').value = 'any';
  // Start the request before clearing, then let its successful response arrive.
  await Promise.resolve();
  vm.runInNewContext("clearQueryEvidence('new query')", context);
  pending.resolve({ ok: true, status: 200, json: async () => ({
    total_count: 9, total_count_is_estimate: false, groups: {},
    state: 'complete', bitmagnet_queries: [], searxng_queries: [],
  }) });
  await invoke;
  const target = widget === 'native facets' ? 'native-facet-status' : 'sweep-status';
  assert.equal(elements.get(target).textContent, 'new query', `${widget} response arriving late must stay cleared`);
}

await proveLateResponseCannotRestore(
  'native facets', 'async function loadNativeFacets()', "\n$('native-facet-button').addEventListener", '',
);
await proveLateResponseCannotRestore(
  'sweep', 'async function sweepAction(execute)', "\n$('sweep-plan').addEventListener", '',
);
const limitsHandler = extract("$('sweep-limits').addEventListener('click',async()=>{", '\n});\n\ncapabilities();');
{
  const elements = makeElements();
  const pending = deferred();
  const context = { $: id => elements.get(id), fetch: () => pending.promise };
  vm.runInNewContext(`let observedToken=0,facetToken=0,sweepToken=0,limitsToken=0; ${clearHelper}; ${limitsHandler}\n});`, context);
  const invoke = elements.get('sweep-limits').listeners.click();
  vm.runInNewContext("clearQueryEvidence('new query')", context);
  pending.resolve({ ok: true, json: async () => ({ remaining_searxng_json_requests: 1 }) });
  await invoke;
  assert.equal(elements.get('sweep-status').textContent, 'new query', 'late limits response must stay cleared');
}

console.log('PASS: query clears invalidate pending facet, sweep, and limits evidence; late facet/sweep responses cannot restore stale output');
