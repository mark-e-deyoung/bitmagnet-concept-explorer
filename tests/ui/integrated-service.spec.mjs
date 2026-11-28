import {test, expect} from "@playwright/test";
import {spawnSync} from "node:child_process";
import AxeBuilder from "@axe-core/playwright";

test("real Go UI retrieves synthetic Bitmagnet data without semantic promotion", async ({page, request}) => {
  const caps = await request.get("/api/v1/capabilities");
  expect(caps.ok()).toBe(true);
  const capability = await caps.json();
  expect(capability.execute).toBe(true);
  expect(capability.query_plan).toBe(true);
  await page.goto("/", {waitUntil: "domcontentloaded"});
  const catalog = await (await request.get("/api/v1/concepts")).json();
  expect(catalog.concepts).toHaveLength(7);
  // The human relationship view intentionally shows the selected focus neighborhood,
  // not every node in the complete catalog. The root fixture has three outgoing edges.
  await expect(page.locator("#concept-list button")).toHaveCount(4);
  await expect(page.locator("#concept-matches button")).toHaveCount(7);
  await page.locator("#query").fill("Animation");
  await page.locator("#concept-filter").fill("book");
  await expect(page.locator("#concept-matches button")).toHaveCount(2);
  await expect(page.locator("#query")).toHaveValue("Animation");
  const matchingApi=await (await request.get("/api/v1/concepts?q=book")).json();
  const matchingUI=await page.locator("#concept-matches button").evaluateAll(nodes=>nodes.map(node=>node.dataset.conceptId));
  expect(matchingUI).toEqual(matchingApi.concepts.map(node=>node.id));
  await page.locator("#concept-filter").fill("หนังสือ");
  await expect(page.locator("#concept-matches button")).toHaveCount(1);
  await expect(page.locator("#query")).toHaveValue("Animation");
  await page.locator("#submit").click();
  await expect(page.locator("#status")).toHaveAttribute("data-ui-state", "ready");
  await expect(page.locator("#status")).toContainText("Unique observed: 1");
  await expect(page.locator("#results")).toContainText("Synthetic Integrated Animation Result");
  const scan = await new AxeBuilder({page}).withTags(["wcag2a","wcag2aa","wcag21a","wcag21aa","wcag22a","wcag22aa"]).analyze();
  expect(scan.violations, scan.violations.map(v=>v.id+": "+v.description).join("; ")).toEqual([]);
  const excess = await page.evaluate(() => Math.max(0,document.documentElement.scrollWidth-document.documentElement.clientWidth));
  expect(excess,"Real Go UI must reflow on this browser viewport").toBeLessThanOrEqual(1);
});

test("automated CLI and agent API agree on actual Go concept graph", async ({request, baseURL}) => {
  const res = await request.get("/api/v1/concepts?q=Medien");
  expect(res.status()).toBe(200);
  const graph = await res.json();
  expect(graph.authority_status).toBe("pinned-preview-not-normative");
  expect(graph.concepts.map(c => c.id)).toEqual(["fixture.media"]);
  const exe = process.env.BCE_CLI_BIN;
  expect(exe, "CI must build actual Go CLI").toBeTruthy();
  const result = spawnSync(exe, ["-server", baseURL, "-operation", "concepts", "-query", "Medien"], {
    encoding: "utf-8", timeout: 12000, env: {...process.env, BCE_AUTH_USER: "", BCE_AUTH_PASSWORD: ""}
  });
  expect(result.error).toBeUndefined();
  expect(result.status, result.stderr).toBe(0);
  expect(JSON.parse(result.stdout)).toEqual(graph);
});

test("no matching provisional terms remains separate from CPE acceptance", async ({request}) => {
  const res = await request.get("/api/v1/concepts?q=unlisted");
  expect(res.status()).toBe(200);
  const graph = await res.json();
  expect(graph.authority_status).toBe("pinned-preview-not-normative");
  expect(graph.concepts).toEqual([]);
  expect(graph.warnings.join(" ")).toContain("does not certify CPE semantic acceptance");
});


test("real Go plan API and CLI produce equal bounded read-only plans", async ({request, baseURL}) => {
  const args = ["-server", baseURL, "-operation", "plan", "-query", "Animation"];
  const planInput = {
    query: "Animation", expansions: [], concept_ids: null,
    filters: {adult_metadata: "any"}, sort: "info_hash", group_by: "none",
    include_web: false
  };
  const api = await request.post("/api/v1/query/plan", {data: planInput});
  expect(api.status(), await api.text()).toBe(200);
  const apiPlan = await api.json();
  const cli = spawnSync(process.env.BCE_CLI_BIN, args, {
    encoding: "utf-8", timeout: 12000,
    env: {...process.env, BCE_AUTH_USER: "", BCE_AUTH_PASSWORD: ""}
  });
  expect(cli.error).toBeUndefined();
  expect(cli.status, cli.stderr).toBe(0);
  const cliPlan = JSON.parse(cli.stdout);
  expect(cliPlan).toEqual(apiPlan);
  expect(apiPlan.read_only).toBe(true);
  expect(apiPlan.tracks[0].name).toBe("direct");
  expect(apiPlan.tracks[0].terms).toEqual(["Animation"]);
  expect(apiPlan.warnings.join(" ")).toContain("not normative CPE authority");
});

test("real Go execute API and CLI retain equivalent observed results without web egress", async ({request, baseURL}) => {
  const input = {
    query: "Animation", expansions: [], concept_ids: null,
    filters: {adult_metadata: "any"}, sort: "info_hash", group_by: "none",
    include_web: false
  };
  const api = await request.post("/api/v1/query/execute", {data: input});
  expect(api.status(), await api.text()).toBe(200);
  const apiResult = await api.json();
  const cli = spawnSync(process.env.BCE_CLI_BIN,
    ["-server", baseURL, "-operation", "execute", "-query", "Animation"], {
      encoding: "utf-8", timeout: 12000,
      env: {...process.env, BCE_AUTH_USER: "", BCE_AUTH_PASSWORD: ""}
    });
  expect(cli.error).toBeUndefined();
  expect(cli.status, cli.stderr).toBe(0);
  const cliResult = JSON.parse(cli.stdout);
  const stable = result => ({
    state: result.state,
    observations: (result.results || []).map(item => ({
      title: item.title, info_hash: item.info_hash,
      tracks: [...(item.tracks || [])].sort()
    })).sort((a, b) => a.title.localeCompare(b.title)),
    counts: result.track_counts
  });
  const a = stable(apiResult), b = stable(cliResult);
  expect(a).toEqual(b);
  expect(a.observations).toHaveLength(1);
  expect(a.observations[0].title).toBe("Synthetic Integrated Animation Result");
  expect(a.observations[0].tracks).toContain("direct");
  expect(apiResult.external_results || []).toHaveLength(0);
  expect(cliResult.external_results || []).toHaveLength(0);
});


test("compiled Go feedback preview API and CLI preserve identical bounded local evidence", async ({request, baseURL}) => {
  const payload = {
    query: {
      query: "Animation", include_web: false,
      expansions: [{term: "Entwurf", provenance: "manual-unverified"}],
      concept_ids: ["fixture.media"],
      filters: {languages: ["en", "de"], content_types: ["movie"], adult_metadata: "no", min_seeders: 2, min_leechers: 0},
      sort: "seeders_desc", group_by: "track"
    },
    proposal_kind: "lexicalization_candidate",
    candidate_label: "fictional lexical hypothesis",
    language: "en"
  };
  const api = await request.post("/api/v1/discovery/feedback/preview", {data: payload});
  expect(api.status(), await api.text()).toBe(200);
  const apiResult = await api.json();
  const cli = spawnSync(process.env.BCE_CLI_BIN,
    ["-server", baseURL, "-operation", "feedback-preview", "-input", "-"], {
      input: JSON.stringify(payload), encoding: "utf-8", timeout: 12000,
      env: {...process.env, BCE_AUTH_USER: "", BCE_AUTH_PASSWORD: ""}
    });
  expect(cli.error).toBeUndefined();
  expect(cli.status, cli.stderr).toBe(0);
  const cliResult = JSON.parse(cli.stdout);
  expect(cliResult).toEqual(apiResult);
  expect(apiResult.state).toBe("local_review_export_only");
  expect(apiResult.observation_count).toBe(1);
  expect(apiResult.proposal.authority).toBe("not_accepted_not_transmitted");
  expect(apiResult.proposal.distinct_ancestry_count).toBe(1);
  expect(apiResult.proposal.evidence).toHaveLength(1);
  expect(JSON.stringify(cliResult)).not.toContain("11111111111111111111");
  expect(JSON.stringify(cliResult)).not.toContain("Synthetic Integrated Animation Result");
  expect(JSON.stringify(cliResult)).not.toContain('"query":"Animation"');
  const denied = {...payload, query: {query: "Animation", include_web: true}};
  const deniedApi = await request.post("/api/v1/discovery/feedback/preview", {data: denied});
  expect(deniedApi.status()).toBe(422);
  const deniedCLI = spawnSync(process.env.BCE_CLI_BIN,
    ["-server", baseURL, "-operation", "feedback-preview", "-input", "-"], {
      input: JSON.stringify(denied), encoding: "utf-8", timeout: 12000,
      env: {...process.env, BCE_AUTH_USER: "", BCE_AUTH_PASSWORD: ""}
    });
  expect(deniedCLI.status).toBe(4);
  expect(JSON.parse(deniedCLI.stdout)).toEqual(await deniedApi.json());
});

test("compiled Go media lookup API and CLI agree on disabled external and unconfigured local providers", async ({request, baseURL}) => {
  const payload = {
    kind: "movie", title: "Fictional Film", providers: ["tmdb", "jellyfin", "imdb"],
    include_related: false, allow_external_lookup: false
  };
  const api = await request.post("/api/v1/media/lookup", {data: payload});
  expect(api.status(), await api.text()).toBe(200);
  const apiResult = await api.json();
  const cli = spawnSync(process.env.BCE_CLI_BIN,
    ["-server", baseURL, "-operation", "media-lookup", "-input", "-"], {
      input: JSON.stringify(payload), encoding: "utf-8", timeout: 12000,
      env: {...process.env, BCE_AUTH_USER: "", BCE_AUTH_PASSWORD: ""}
    });
  expect(cli.error).toBeUndefined();
  expect(cli.status, cli.stderr).toBe(0);
  expect(JSON.parse(cli.stdout)).toEqual(apiResult);
  expect(apiResult.read_only).toBe(true);
  expect(apiResult.outcomes.map(v => v.state)).toEqual([
    "requires_explicit_external_consent",
    "unconfigured_local_library",
    "licensed_source_not_configured"
  ]);
  expect(apiResult.outcomes.every(v => v.candidates.length === 0 && v.related.length === 0)).toBe(true);
});


test("human media and concept feedback controls call actual Go routes without source egress", async ({page}) => {
  const mediaRequests = [];
  page.on("request", request => { if (request.url().endsWith("/api/v1/media/lookup")) mediaRequests.push(request); });
  await page.goto("/", {waitUntil: "domcontentloaded"});
  await page.locator("#query").fill("Animation");
  await page.locator("#submit").click();
  await expect(page.locator("#status")).toHaveAttribute("data-ui-state", "ready");
  await expect(page.locator(".media-inspect")).toHaveCount(1);
  await page.locator(".media-inspect").first().click();
  await expect(page.locator("#lookup-title")).toHaveValue("Synthetic Integrated Animation Result");
  await expect(page.locator("#lookup-consent")).not.toBeChecked();
  await page.locator("#lookup-provider").selectOption("tmdb");
  await page.locator("#lookup-submit").click();
  await expect(page.locator("#lookup-status")).toHaveAttribute("data-ui-state","blocked");
  await expect(page.locator("#lookup-consent")).toHaveAttribute("aria-invalid", "true");
  await expect(page.locator("#lookup-consent")).toHaveAttribute("aria-describedby", /lookup-status/);
  await expect(page.locator("#lookup-consent")).toBeFocused();
  expect(mediaRequests, "external source lookup needs fresh consent before even contacting the local lookup route").toHaveLength(0);

  await page.locator("#feedback-candidate").fill("fictional lexical form");
  await page.locator("#feedback-language").fill("de");
  await page.locator("#feedback-kind").selectOption("lexicalization_candidate");
  await page.locator("#feedback-submit").click();
  await expect(page.locator("#feedback-status")).toHaveAttribute("data-ui-state","ready");
  await expect(page.locator("#feedback-status")).toContainText("not sent to or accepted by CPE");
  await expect(page.locator("#feedback-output")).toContainText("not_accepted_not_transmitted");
  await expect(page.locator("#feedback-output")).not.toContainText("1111111111111111111111111111111111111111");
});

test("human feedback preserves the submitted local query scope and never inherits web consent", async ({page, baseURL}) => {
  const posts = [];
  const offOrigin = [];
  page.on("request", request => {
    if (request.method() === "POST") posts.push(request);
    if (new URL(request.url()).origin !== new URL(baseURL).origin) offOrigin.push(request.url());
  });
  await page.goto("/", {waitUntil: "domcontentloaded"});
  await page.locator('#concept-matches button[data-concept-id="fixture.media"]').click();
  await page.locator("#use-concept").check();
  await page.locator("#query").fill("Animation");
  await page.locator("#advanced-settings summary").click();
  await page.locator("#terms").fill(" Entwurf \n เทศกาล ");
  await page.locator("#languages").fill("en, de");
  await page.locator("#media").selectOption("movie");
  await page.locator("#adult").selectOption("no");
  await page.locator("#minseeders").fill("2");
  await page.locator("#minleechers").fill("0");
  await page.locator("#sort").selectOption("seeders_desc");
  await page.locator("#group").selectOption("track");
  expect(posts, "editing local scope must not initiate retrieval").toHaveLength(0);
  const queryRequest = page.waitForRequest(request => request.url().endsWith("/api/v1/query/execute") && request.method() === "POST");
  await page.locator("#submit").click();
  const submittedScope = (await queryRequest).postDataJSON();
  await expect(page.locator("#status")).toHaveAttribute("data-ui-state", "ready");
  expect(submittedScope).toEqual({
    query: "Animation",
    expansions: [{term: "Entwurf", provenance: "manual-unverified"}, {term: "เทศกาล", provenance: "manual-unverified"}],
    concept_ids: ["fixture.media"],
    filters: {languages: ["en", "de"], content_types: ["movie"], adult_metadata: "no", min_seeders: 2, min_leechers: 0},
    sort: "seeders_desc", group_by: "track", include_web: false
  });
  // Ordinary web consent is specific to its workflow. A feedback preview is
  // always local even when this separate checkbox is checked.
  await page.locator("#web").check();
  await page.locator("#feedback-candidate").fill("Synthetic lexical hypothesis");
  await page.locator("#feedback-language").fill("de");
  const previewRequest = page.waitForRequest(request => request.url().endsWith("/api/v1/discovery/feedback/preview") && request.method() === "POST");
  await page.locator("#feedback-submit").click();
  const preview = (await previewRequest).postDataJSON();
  expect(preview.query).toEqual({...submittedScope, include_web: false});
  await expect(page.locator("#feedback-status")).toHaveAttribute("data-ui-state", "ready");
  await expect(page.locator("#feedback-output")).toContainText("not_accepted_not_transmitted");
  await expect(page.locator("#feedback-submit")).toBeEnabled();
  expect(offOrigin, "browser requests must stay inside this isolated service fixture").toEqual([]);
});

function pendingGate() {
  let resolve;
  const promise = new Promise(done => { resolve = done; });
  return {promise, resolve};
}

// Deliberately hold only the first synthetic sweep response; query execution
// and the replacement sweep still use the actual compiled Go service.
async function holdFirstSweep(page) {
  const release = pendingGate();
  const started = pendingGate();
  const settled = pendingGate();
  let count = 0;
  await page.route("**/api/v1/sweeps/plan", async route => {
    if (++count > 1) { await route.continue(); return; }
    started.resolve();
    await release.promise;
    try {
      await route.fulfill({status: 200, contentType: "application/json", body: JSON.stringify({state: "planned", bitmagnet_queries: ["stale synthetic query"], searxng_queries: []})});
    } catch (error) {
      // The browser may have already canceled this intercepted request. Other
      // fixture failures must remain visible instead of becoming silent passes.
      if (!route.request().failure()) throw error;
    } finally { settled.resolve(); }
  });
  return {...release, started: started.promise, settled: settled.promise};
}

test("ordinary query supersedes a pending sweep and immediately restores usable controls", async ({page}) => {
  const release = await holdFirstSweep(page);
  try {
    await page.goto("/", {waitUntil: "domcontentloaded"});
    await page.locator("#query").fill("Animation");
    await page.locator("#sweep-plan").click();
    await release.started;
    await expect(page.locator("#sweep-run")).toBeDisabled();
    await expect(page.locator("#sweep-plan")).toBeDisabled();
    await page.locator("#submit").click();
    await expect(page.locator("#sweep-run")).toBeEnabled();
    await expect(page.locator("#sweep-plan")).toBeEnabled();
    await expect(page.locator("#status")).toHaveAttribute("data-ui-state", "ready");
    release.resolve();
    await release.settled;
    await expect(page.locator("#sweep-details")).not.toContainText("stale synthetic query");
    await page.locator("#sweep-plan").click();
    await expect(page.locator("#sweep-status")).toHaveAttribute("data-ui-state", "ready");
    await expect(page.locator("#sweep-plan")).toBeEnabled();
  } finally { release.resolve(); }
});

test("canceling a pending sweep restores controls and permits a fresh local plan", async ({page}) => {
  const release = await holdFirstSweep(page);
  try {
    await page.goto("/", {waitUntil: "domcontentloaded"});
    await page.locator("#query").fill("Animation");
    await page.locator("#sweep-plan").click();
    await release.started;
    await expect(page.locator("#sweep-cancel")).toBeEnabled();
    await page.locator("#sweep-cancel").click();
    await expect(page.locator("#sweep-run")).toBeEnabled();
    await expect(page.locator("#sweep-plan")).toBeEnabled();
    await expect(page.locator("#sweep-status")).toContainText(/cancel/i);
    release.resolve();
    await release.settled;
    await page.locator("#sweep-plan").click();
    await expect(page.locator("#sweep-status")).toHaveAttribute("data-ui-state", "ready");
    await expect(page.locator("#sweep-details")).not.toContainText("stale synthetic query");
  } finally { release.resolve(); }
});

test("changing request scope clears external consent without automatically making a request", async ({page}) => {
  const posts = [];
  page.on("request", request => { if (request.method() === "POST") posts.push(request.url()); });
  await page.goto("/", {waitUntil: "domcontentloaded"});
  await page.locator("#advanced-settings summary").click();
  await page.locator("#query").fill("Animation");
  await page.locator("#web").check();
  await page.locator("#sweep-web").check();
  await page.locator("#sweep-consent").check();
  await page.locator("#query").fill("Changed synthetic phrase");
  await expect(page.locator("#web")).not.toBeChecked();
  await expect(page.locator("#sweep-web")).not.toBeChecked();
  await expect(page.locator("#sweep-consent")).not.toBeChecked();
  await page.locator("#lookup-provider").selectOption("tmdb");
  await page.locator("#lookup-title").fill("Synthetic title");
  await page.locator("#lookup-consent").check();
  await page.locator("#lookup-title").fill("Changed synthetic title");
  await expect(page.locator("#lookup-consent")).not.toBeChecked();
  expect(posts, "consent and scope changes must not initiate retrieval").toEqual([]);
});

test("compiled Go metadata comparison and CLI agree without external calls", async ({request, baseURL}) => {
  const payload = {
    reference: {title:"Fictional Example",kind:"movie",year:2025,catalog_ids:{imdb:"tt1234567"}},
    observations: [
      {source:"searxng",state:"observed",record:{title:"Fictional Example"}},
      {source:"reference_catalog",state:"observed",record:{title:"Fictional Example",year:2024,catalog_ids:{imdb:"tt1234567"}}},
      {source:"publisher",state:"refused",record:{}}
    ]
  };
  const api = await request.post("/api/v1/metadata/compare",{data:payload});
  expect(api.status(),await api.text()).toBe(200);
  const result=await api.json();
  const cli=spawnSync(process.env.BCE_CLI_BIN,["-server",baseURL,"-operation","metadata-compare","-input","-"],{
    input:JSON.stringify(payload),encoding:"utf-8",timeout:12000,
    env:{...process.env,BCE_AUTH_USER:"",BCE_AUTH_PASSWORD:""}
  });
  expect(cli.error).toBeUndefined();
  expect(cli.status,cli.stderr).toBe(0);
  expect(JSON.parse(cli.stdout)).toEqual(result);
  expect(result.read_only).toBe(true);
  expect(result.external_requests).toBe(0);
  expect(result.independent_witnesses).toBe(0);
  expect(result.outcomes.map(x=>x.identity)).toEqual(["title_candidate_unconfirmed","conflicting_claims_review_required","source_incomplete_non_diagnostic"]);
  expect(cli.stdout).not.toContain("Fictional Example");
  expect(cli.stdout).not.toContain("tt1234567");
});
