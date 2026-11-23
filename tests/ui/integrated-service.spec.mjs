import {test, expect} from "@playwright/test";
import {spawnSync} from "node:child_process";

test("real Go UI retrieves synthetic Bitmagnet data without semantic promotion", async ({page, request}) => {
  const caps = await request.get("/api/v1/capabilities");
  expect(caps.ok()).toBe(true);
  const capability = await caps.json();
  expect(capability.execute).toBe(true);
  expect(capability.query_plan).toBe(true);
  await page.goto("/", {waitUntil: "domcontentloaded"});
  await expect(page.locator("#concept-list button")).toHaveCount(7);
  await page.locator("#query").fill("Animation");
  await page.locator("#submit").click();
  await expect(page.locator("#status")).toHaveAttribute("data-ui-state", "ready");
  await expect(page.locator("#status")).toContainText("Unique observed: 1");
  await expect(page.locator("#results")).toContainText("Synthetic Integrated Animation Result");
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
