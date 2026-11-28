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
const sharedHelpers = extract('function setStatus(', '\nfunction displayItems(')
  + extract('function cancelSweep(', '\nfunction buildQueryRequest(') + clearHelper;
const commonState = 'let observedToken=0,facetToken=0,sweepToken=0,limitsToken=0,sweepDisplayToken=0,canExecute=true,queryToken=0,activeQueryController=null,activeSweepController=null;';
const clearIDs = [
  'status', 'direct', 'concept', 'overlap', 'external', 'details',
  'observed-graph', 'observed-list', 'observed-items', 'observed-status',
  'native-facet-data', 'native-facet-status', 'sweep-details', 'sweep-status',
];
const makeElements = () => {
  const document = {activeElement: null};
  const elements = new Map([...clearIDs, 'minseeders', 'minleechers', 'adult', 'sweep-run', 'sweep-plan', 'sweep-limits',
    'sweep-cancel', 'sweep-web', 'sweep-consent', 'languages', 'media', 'query', 'query-cancel', 'submit',
    'sweep-local-budget', 'sweep-external-budget', 'results', 'result-scope', 'terms', 'sort', 'group', 'web', 'use-concept', 'consent-status', 'q', 'sweep'].map(id => [id, {
    id, textContent: 'stale evidence from previous query', children: [], value: '', disabled: false, checked: false,
    attributes: new Map(), dataset: {}, listeners: {}, replaceChildren() { this.children = []; },
    setAttribute(name, value) { this.attributes.set(name, String(value)); },
    getAttribute(name) { return this.attributes.get(name) ?? null; },
    removeAttribute(name) { this.attributes.delete(name); },
    querySelectorAll(selector) {
      assert.equal(selector, '[aria-invalid="true"]', 'fixture supports the actual invalid-field selector');
      const ids = this.id === 'q'
        ? ['query', 'terms', 'languages', 'media', 'minseeders', 'minleechers', 'adult', 'sort', 'group', 'web']
        : [...elements.keys()].filter(key => key.startsWith('sweep-'));
      return ids.map(key => elements.get(key)).filter(field => field.getAttribute('aria-invalid') === 'true');
    },
    closest() { return null; },
    focus() { document.activeElement = this; },
    addEventListener(name, listener) { this.listeners[name] = listener; },
  }]));
  elements.get('sweep-local-budget').value = '3';
  elements.get('sweep-external-budget').value = '1';
  elements.document = document;
  return elements;
};
const makeContext = (elements, additions = {}) => ({
  $: id => elements.get(id), document: elements.document,
  AbortController, setTimeout, clearTimeout,
  ...additions,
});

for (const scenario of ['new request', 'timeout', 'HTTP 422', 'HTTP 401']) {
  const elements = makeElements();
  const context = makeContext(elements);
  const generations = vm.runInNewContext(`${commonState} ${sharedHelpers}; clearQueryEvidence(${JSON.stringify(scenario)}); [observedToken,facetToken,sweepToken,limitsToken,sweepDisplayToken]`, context);
  assert.deepEqual(Array.from(generations), [1, 1, 1, 1, 1]);
  for (const id of clearIDs.filter(id => id !== 'status')) {
    assert.equal(elements.get(id).textContent, id.endsWith('-status') ? '' : scenario, `${scenario} clears ${id}; only the main live status announces the reset`);
    assert.deepEqual(elements.get(id).children, [], `${scenario} removes nodes from ${id}`);
  }
}

function deferred() {
  let resolve, reject;
  const promise = new Promise((yes, no) => { resolve = yes; reject = no; });
  return { promise, resolve, reject };
}

async function proveLateResponseCannotRestore(widget, start, end, setup) {
  const fn = extract(start, end);
  const elements = makeElements();
  const pending = deferred();
  let fetchCalls = 0;
  elements.get('adult').value = 'any';
  const context = makeContext(elements, {
    fetch: () => { fetchCalls++; return pending.promise; },
    buildSweepRequest: () => ({ query: 'synthetic', include_searxng: false, external_consent: false }),
  });
  const invoke = vm.runInNewContext(`${setup}
    ${commonState}
    ${sharedHelpers}
    ${fn}
    ${widget === 'native facets' ? 'loadNativeFacets()' : 'sweepAction(false)'}`, context);
  elements.get('minseeders').value = '';
  elements.get('minleechers').value = '';
  // Start the request before clearing, then let its successful response arrive.
  await Promise.resolve();
  assert.equal(fetchCalls, 1, `${widget} test must start a real deferred request`);
  vm.runInNewContext("clearQueryEvidence('new query')", context);
  if (widget === 'sweep') {
    for (const id of ['sweep-run', 'sweep-plan']) {
      assert.equal(elements.get(id).disabled, false,
        `new query immediately releases ${id}, even before the superseded sweep settles`);
    }
  }
  pending.resolve({ ok: true, status: 200, json: async () => ({
    total_count: 9, total_count_is_estimate: false, groups: {},
    state: 'complete', bitmagnet_queries: [], searxng_queries: [],
  }) });
  await invoke;
  const target = widget === 'native facets' ? 'native-facet-status' : 'sweep-status';
  assert.equal(elements.get(target).textContent, '', `${widget} response arriving late must stay cleared without duplicating the main live status`);
  if (widget === 'sweep') {
    for (const id of ['sweep-run', 'sweep-plan']) {
      assert.equal(elements.get(id).disabled, false,
        `superseded sweep cannot leave ${id} disabled`);
    }
  }
}

await proveLateResponseCannotRestore(
  'native facets', 'async function loadNativeFacets()', "\n$('native-facet-button').addEventListener", '',
);
await proveLateResponseCannotRestore(
  'sweep', 'async function sweepAction(execute)', "\n$('sweep-plan').addEventListener", '',
);
const limitsHandler = extract("$('sweep-limits').addEventListener('click',async()=>{", '\n});\n\n// Example phrases');
{
  const elements = makeElements();
  const pending = deferred();
  const context = makeContext(elements, {fetch: () => pending.promise});
  vm.runInNewContext(`${commonState} ${sharedHelpers}; ${limitsHandler}\n});`, context);
  const invoke = elements.get('sweep-limits').listeners.click();
  vm.runInNewContext("clearQueryEvidence('new query')", context);
  pending.resolve({ ok: true, json: async () => ({ remaining_searxng_json_requests: 1 }) });
  await invoke;
  assert.equal(elements.get('sweep-status').textContent, '', 'late limits response must stay cleared without duplicating the main live status');
}

const sweepFunction = extract('async function sweepAction(execute)', "\n$('sweep-plan').addEventListener");
const limitsResponse = {
  remaining_searxng_json_requests: 2, max_searxng_json_requests_per_hour: 3,
  searxng_cooldown_seconds: 0, sweep_deadline_seconds: 55, client_timeout_seconds: 80,
  max_concurrent_sweeps: 1, searxng_sweep_admitted: true,
};
const sweepResponse = { bitmagnet_queries: [{}], searxng_queries: [], state: 'complete' };
for (const newer of ['limits', 'sweep']) {
  const elements = makeElements();
  const pending = new Map(['/api/v1/sweeps/limits', '/api/v1/sweeps/plan'].map(path => [path, deferred()]));
  const context = makeContext(elements, {
    fetch: path => pending.get(path).promise,
    buildSweepRequest: () => ({ query: 'synthetic', include_searxng: false, external_consent: false }),
  });
  vm.runInNewContext(`${commonState} ${sharedHelpers}; ${sweepFunction}; ${limitsHandler}\n});`, context);
  const sweepPromise = vm.runInNewContext('sweepAction(false)', context);
  const limitsPromise = elements.get('sweep-limits').listeners.click();
  if (newer === 'limits') {
    pending.get('/api/v1/sweeps/plan').resolve({ ok: true, json: async () => sweepResponse });
    await sweepPromise;
    assert.equal(elements.get('sweep-status').textContent, 'Reading local sweep limits…', 'late sweep must not overwrite newer limits status');
    pending.get('/api/v1/sweeps/limits').resolve({ ok: true, json: async () => limitsResponse });
    await limitsPromise;
    assert.match(elements.get('sweep-status').textContent, /^SearXNG JSON budget:/);
  } else {
    // Start a fresh sweep after limits so the sweep owns the shared display.
    const secondSweep = vm.runInNewContext('sweepAction(false)', context);
    pending.get('/api/v1/sweeps/limits').resolve({ ok: true, json: async () => limitsResponse });
    await limitsPromise;
    assert.match(elements.get('sweep-status').textContent, /^Preparing bounded sweep plan/);
    pending.get('/api/v1/sweeps/plan').resolve({ ok: true, json: async () => sweepResponse });
    await Promise.all([sweepPromise, secondSweep]);
    assert.match(elements.get('sweep-status').textContent, /^Plan ready;/);
  }
}

for (const lateOutcome of ['success', 'failure']) {
  const elements = makeElements();
  const pending = deferred();
  let signal;
  const context = makeContext(elements, {
    fetch: (_path, options) => { signal = options.signal; return pending.promise; },
    buildSweepRequest: () => ({query: 'synthetic', concept_ids: [], include_searxng: false, external_consent: false}),
  });
  vm.runInNewContext(`${commonState} ${sharedHelpers}; ${sweepFunction}`, context);
  const request = vm.runInNewContext('sweepAction(false)', context);
  assert.equal(elements.get('sweep-run').disabled, true, 'pending sweep disables repeat execution');
  assert.equal(elements.get('sweep-cancel').disabled, false, 'pending sweep exposes a usable cancel control');
  elements.document.activeElement = elements.get('sweep-cancel');
  vm.runInNewContext("cancelSweep('Canceled by the user')", context);
  assert.equal(signal.aborted, true, 'cancellation aborts the actual pending fetch signal');
  assert.equal(elements.get('sweep-run').disabled, false, 'cancellation immediately restores execution');
  assert.equal(elements.get('sweep-plan').disabled, false, 'cancellation immediately restores planning');
  assert.equal(elements.get('sweep-cancel').disabled, true, 'no active request remains cancelable');
  assert.equal(elements.get('sweep-details').getAttribute('aria-busy'), 'false', 'assistive technology is told the task ended');
  assert.equal(elements.document.activeElement, elements.get('sweep-run'), 'focus moves to an enabled action before disabling cancel');
  // Ignore AbortSignal deliberately: a late transport result must still be
  // guarded by the production generation check, even when abort is too late.
  if (lateOutcome === 'success') pending.resolve({ok: true, json: async () => sweepResponse});
  else pending.reject(new Error('late synthetic transport failure'));
  await request;
  assert.equal(elements.get('sweep-status').textContent, 'Canceled by the user', `late ${lateOutcome} cannot overwrite cancellation`);
  assert.equal(elements.get('sweep-run').disabled, false, `late ${lateOutcome} cannot strand execution controls`);
}

for (const lateOutcome of ['success', 'failure']) {
  const elements = makeElements();
  const first = deferred(), second = deferred();
  const signals = [];
  const context = makeContext(elements, {
    fetch: (_path, options) => { signals.push(options.signal); return signals.length === 1 ? first.promise : second.promise; },
    buildSweepRequest: () => ({query: 'synthetic', concept_ids: [], include_searxng: false, external_consent: false}),
  });
  vm.runInNewContext(`${commonState} ${sharedHelpers}; ${sweepFunction}`, context);
  const firstRequest = vm.runInNewContext('sweepAction(false)', context);
  vm.runInNewContext("clearQueryEvidence('Changed query')", context);
  const secondRequest = vm.runInNewContext('sweepAction(false)', context);
  assert.equal(signals[0].aborted, true, 'changing query aborts the old sweep');
  assert.equal(signals[1].aborted, false, 'new request retains its own active signal');
  if (lateOutcome === 'success') first.resolve({ok: true, json: async () => ({...sweepResponse, bitmagnet_queries: ['old result']})});
  else first.reject(new Error('late synthetic transport failure'));
  await firstRequest;
  assert.match(elements.get('sweep-status').textContent, /^Preparing bounded sweep plan/);
  assert.equal(elements.get('sweep-run').disabled, true, `old ${lateOutcome} must not release the newer request's controls`);
  assert.equal(elements.get('sweep-details').getAttribute('aria-busy'), 'true', 'newer task remains announced as busy');
  second.resolve({ok: true, json: async () => sweepResponse});
  await secondRequest;
  assert.match(elements.get('sweep-status').textContent, /^Plan ready;/);
  assert.equal(elements.get('sweep-run').disabled, false);
  assert.equal(elements.get('sweep-details').getAttribute('aria-busy'), 'false');
}

const querySource = extract('function buildQueryRequest()', '\nfunction clearQueryEvidence(')
  + extract('function displayItems(', '\nfunction cancelSweep(')
  + extract('async function renderObservedGraph(rows,signal)', '\nasync function loadNativeFacets()')
  + extract("$('q').addEventListener('submit',async event=>{", '\n// The graph is read-only');
for (const lateOutcome of ['success', 'failure']) {
  const elements = makeElements();
  elements.get('query').value = 'Preserved synthetic phrase';
  elements.get('terms').value = 'Preserved expansion';
  elements.get('adult').value = 'any';
  elements.get('sort').value = 'info_hash';
  elements.get('group').value = 'none';
  const first = deferred(), second = deferred(), signals = [];
  const context = makeContext(elements, {
    fetch: (_path, options) => { signals.push(options.signal); return signals.length === 1 ? first.promise : second.promise; },
  });
  vm.runInNewContext(`${commonState} let focusConcept=null; ${sharedHelpers}; ${querySource}`, context);
  const firstRequest = elements.get('q').listeners.submit({preventDefault() {}});
  assert.equal(elements.get('submit').disabled, true);
  assert.equal(elements.get('query-cancel').disabled, false);
  assert.equal(elements.get('results').getAttribute('aria-busy'), 'true');
  elements.document.activeElement = elements.get('query-cancel');
  vm.runInNewContext('cancelQuery()', context);
  assert.equal(signals[0].aborted, true, 'search cancellation aborts its own pending fetch');
  assert.equal(elements.get('submit').disabled, false, 'search cancellation immediately permits a retry');
  assert.equal(elements.get('query-cancel').disabled, true);
  assert.equal(elements.get('results').getAttribute('aria-busy'), 'false');
  assert.equal(elements.document.activeElement, elements.get('submit'), 'search cancellation restores an enabled focus target');
  assert.equal(elements.get('query').value, 'Preserved synthetic phrase');
  assert.equal(elements.get('terms').value, 'Preserved expansion');
  const secondRequest = elements.get('q').listeners.submit({preventDefault() {}});
  const response = {ok: true, json: async () => ({state: 'complete', results: [], track_counts: {direct: 0, concept: 0}})};
  if (lateOutcome === 'success') first.resolve(response);
  else first.reject(new Error('late synthetic search failure'));
  await firstRequest;
  assert.equal(elements.get('status').getAttribute('data-ui-state'), 'busy', `late search ${lateOutcome} cannot overwrite a newer request`);
  assert.equal(elements.get('submit').disabled, true, 'old search cannot release the active search controls');
  assert.equal(elements.get('results').getAttribute('aria-busy'), 'true');
  second.resolve(response);
  await secondRequest;
  assert.equal(elements.get('status').getAttribute('data-ui-state'), 'ready');
  assert.equal(elements.get('submit').disabled, false);
  assert.equal(elements.get('results').getAttribute('aria-busy'), 'false');
}

console.log('PASS: query/sweep cancellation aborts pending requests, restores controls and focus, preserves inputs, and rejects stale success/failure without releasing newer work; facet/limits evidence stays invalidated');
