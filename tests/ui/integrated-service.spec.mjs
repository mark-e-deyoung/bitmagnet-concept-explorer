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
  await page.locator("#query").fill("Animation");
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
