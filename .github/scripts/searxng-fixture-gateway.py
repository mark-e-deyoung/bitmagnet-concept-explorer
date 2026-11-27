#!/usr/bin/env python3
"""Disposable CI fixture gateway; not a general search proxy or deployment."""
import hmac
import http.client
import json
import os
import threading
import urllib.parse
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

TOKEN = os.environ["RDTE_TOKEN"]
FIXTURES = {
    "/probe/en": ("en", "water cycle evaporation condensation precipitation"),
    "/probe/de": ("de", "Wasserkreislauf Verdunstung Kondensation Niederschlag"),
    "/probe/th": ("th", "วัฏจักรน้ำ การระเหย การควบแน่น"),
}
LOCK = threading.Lock()
USED = set()

class Handler(BaseHTTPRequestHandler):
    def log_message(self, *_):
        pass

    def reply(self, code, body):
        data = json.dumps(body, ensure_ascii=True).encode("utf-8")
        self.send_response(code)
        self.send_header("Content-Type", "application/json")
        self.send_header("Cache-Control", "no-store")
        self.send_header("Content-Length", str(len(data)))
        self.end_headers()
        self.wfile.write(data)

    def do_GET(self):
        if not hmac.compare_digest(self.headers.get("X-RDTE-Token", ""), TOKEN):
            return self.reply(401, {"error": "fixture_auth_required"})
        if self.path == "/healthz":
            return self.reply(200, {"fixture_only": True})
        if self.path not in FIXTURES:
            return self.reply(404, {"error": "fixed_fixture_route_only"})
        with LOCK:
            if self.path in USED:
                return self.reply(429, {"error": "one_probe_per_fixture"})
            USED.add(self.path)
        lang, query = FIXTURES[self.path]
        params = urllib.parse.urlencode({
            "q": query, "language": lang, "format": "json",
            "engines": "wikipedia,duckduckgo,bing", "pageno": 1,
            "safesearch": 1,
        })
        conn = http.client.HTTPConnection("127.0.0.1", 18888, timeout=35)
        try:
            conn.request("POST", "/search", body=params, headers={
                "Content-Type": "application/x-www-form-urlencoded",
                "Accept": "application/json",
            })
            res = conn.getresponse()
            raw = res.read(2_000_001)
            if res.status != 200 or len(raw) > 2_000_000:
                return self.reply(502, {"error": "upstream_status_or_budget",
                                        "upstream_status": res.status, "language": lang})
            obj = json.loads(raw)
            hits = obj.get("results", [])
            if not isinstance(hits, list):
                raise ValueError("unexpected result schema")
            engines = sorted({str(e)[:80] for x in hits if isinstance(x, dict)
                              for e in x.get("engines", []) if isinstance(e, str)})
            failures = obj.get("unresponsive_engines", [])
            self.reply(200, {"language_requested": lang,
                             "result_count": len(hits),
                             "engines_observed": engines,
                             "unresponsive_engine_count": len(failures),
                             "fixture_only": True})
        except (OSError, ValueError, http.client.HTTPException):
            self.reply(502, {"error": "upstream_failed", "language_requested": lang})
        finally:
            conn.close()

    def do_POST(self):
        self.reply(405, {"error": "fixed_get_only"})

if __name__ == "__main__":
    ThreadingHTTPServer(("127.0.0.1", 18889), Handler).serve_forever()
