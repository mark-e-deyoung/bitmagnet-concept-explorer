// Public synthetic localhost upstream + actual compiled Go Explorer service.
// Playwright drives its embedded production UI, HTTP API and built CLI.
import http from "node:http";
import { spawn } from "node:child_process";
import { createHash } from "node:crypto";
import { readFileSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const ROOT = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../..");
const BIN = process.env.BCE_SERVICE_BIN;
if (!BIN || !path.isAbsolute(BIN)) throw new Error("BCE_SERVICE_BIN must be an absolute path to the freshly built local Go executable");
const fixturePath = path.join(ROOT, "fixtures/concept-graph.synthetic.json");
const hash = createHash("sha256").update(readFileSync(fixturePath)).digest("hex");
const fakeResult = {
  data: { torrentContent: { search: {
    hasNextPage: false,
    items: [{
      title: "Synthetic Integrated Animation Result",
      contentType: "movie",
      seeders: 12,
      leechers: 3,
      languages: [{ id: "en", name: "English" }],
      content: { adult: false },
      torrent: { infoHash: "1111111111111111111111111111111111111111", name: "synthetic-animation", size: 1024 }
    }]
  } } }
};
const upstream = http.createServer((req, res) => {
  if (req.method !== "POST" || req.url !== "/graphql") {
    res.writeHead(404).end();
    return;
  }
  let size = 0;
  let body = "";
  req.on("data", chunk => {
    size += chunk.length;
    if (size > 65536) req.destroy();
    else body += chunk;
  });
  req.on("end", () => {
    try {
      const payload = JSON.parse(body);
      if (!payload.query || typeof payload.query !== "string") throw new Error("not a GraphQL query");
      res.writeHead(200, { "content-type": "application/json" });
      res.end(JSON.stringify(fakeResult));
    } catch {
      res.writeHead(400, { "content-type": "application/json" });
      res.end('{"errors":[{"message":"synthetic malformed query"}]}');
    }
  });
});
await new Promise((resolve, reject) => {
  upstream.once("error", reject);
  upstream.listen(4179, "127.0.0.1", resolve);
});
const child = spawn(BIN, ["-listen", "127.0.0.1:4178"], {
  cwd: ROOT,
  env: {
    ...process.env,
    BCE_BITMAGNET_GRAPHQL: "http://127.0.0.1:4179/graphql",
    BCE_CPE_GRAPH_FILE: fixturePath,
    BCE_CPE_GRAPH_SHA256: hash,
    BCE_SEARXNG_URL: "",
    BCE_DISCOVERY_HMAC_KEY: "synthetic-only-replay-key-0123456789012345",
    BCE_TMDB_READ_TOKEN: "",
    BCE_JELLYFIN_BASE_URL: "",
    BCE_JELLYFIN_TOKEN: "",
    BCE_AUTH_USER: "",
    BCE_AUTH_PASSWORD: ""
  },
  stdio: ["ignore", "pipe", "pipe"]
});
child.stdout.on("data", chunk => process.stdout.write(chunk));
child.stderr.on("data", chunk => process.stderr.write(chunk));
child.once("exit", code => {
  console.error("Integrated Go service exited: " + code);
  upstream.close();
  process.exitCode = code || 1;
});
function stop() {
  child.kill("SIGTERM");
  upstream.close();
}
process.on("SIGTERM", stop);
process.on("SIGINT", stop);
console.log("Public synthetic source ready; waiting on Go service health 4178");
