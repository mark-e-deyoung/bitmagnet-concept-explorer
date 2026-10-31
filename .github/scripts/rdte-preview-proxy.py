#!/usr/bin/env python3
"""One-shot public-fixture-only, bounded read-only preview proxy.

Ephemeral GitHub Actions test harness only: NEVER deploy as a production
proxy. Origin has demo-only credentials and listens on localhost. Cloudflare
exposes this filter, not Bitmagnet GraphQL/import, DB, or raw origin service.
"""
import base64
import collections
import http.client
import os
import threading
import time
import urllib.error
import urllib.parse
import urllib.request
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

ADDRESS = ("127.0.0.1", 3087)
ORIGIN = "http://127.0.0.1:3086"
USER = os.environ["BCE_PREVIEW_UPSTREAM_USER"]
PASSWORD = os.environ["BCE_PREVIEW_UPSTREAM_PASSWORD"]
AUTH = "Basic " + base64.b64encode((USER + ":" + PASSWORD).encode()).decode()
MAX_BODY = 16 * 1024
MAX_RESPONSE = 2 * 1024 * 1024
SLOTS = threading.BoundedSemaphore(value=4)
REQUESTS = collections.deque()
LOCK = threading.Lock()
GET_PREFIXES = ("/api/v1/",)
POST_PATHS = frozenset({
    "/api/v1/query/plan", "/api/v1/query/execute",
    "/api/v1/query/batch", "/api/v1/query/facets",
    "/api/v1/query/observed-graph", "/api/v1/sweeps/plan",
    "/api/v1/sweeps/execute", "/api/v1/cpe/preflight",
    "/api/v1/query/discovery-preview", "/api/v1/concepts/infer",
})


class Handler(BaseHTTPRequestHandler):
    protocol_version = "HTTP/1.1"

    def log_message(self, fmt, *args):
        # No query/body/credential logging in public Actions job output.
        return

    def reply(self, code, body, kind="text/plain; charset=utf-8"):
        self.send_response(code)
        self.send_header("Content-Type", kind)
        self.send_header("Content-Length", str(len(body)))
        self.send_header("Cache-Control", "no-store")
        self.send_header("X-Content-Type-Options", "nosniff")
        self.end_headers()
        self.wfile.write(body)

    def do_GET(self):
        self.proxy("GET")

    def do_POST(self):
        self.proxy("POST")

    def do_HEAD(self):
        self.reply(405, b"preview does not support HEAD")

    def do_PUT(self):
        self.reply(405, b"preview is read-only")

    def do_PATCH(self):
        self.reply(405, b"preview is read-only")

    def do_DELETE(self):
        self.reply(405, b"preview is read-only")

    def proxy(self, method):
        target = urllib.parse.urlsplit(self.path)
        route = target.path
        if method == "GET":
            permitted = route in ("/", "/healthz") or any(
                route.startswith(prefix) for prefix in GET_PREFIXES
            )
        else:
            permitted = route in POST_PATHS
        if not permitted or target.scheme or target.netloc:
            self.reply(405, b"preview allows documented read-only routes only")
            return
        try:
            length = int(self.headers.get("Content-Length", "0"))
        except ValueError:
            self.reply(400, b"invalid request length")
            return
        if length < 0 or length > MAX_BODY:
            self.reply(413, b"preview request too large")
            return
        now = time.monotonic()
        with LOCK:
            while REQUESTS and now - REQUESTS[0] > 60:
                REQUESTS.popleft()
            if len(REQUESTS) >= 100:
                self.reply(429, b"preview rate limit")
                return
            REQUESTS.append(now)
        if not SLOTS.acquire(blocking=False):
            self.reply(429, b"preview busy")
            return
        try:
            request_data = self.rfile.read(length) if length else None
            # Do not forward client headers, cookies or auth. Use only the
            # local public-fixture upstream identity in the private runner.
            headers = {"Authorization": AUTH, "Accept": "application/json, text/html"}
            if method == "POST":
                headers["Content-Type"] = "application/json"
            url = ORIGIN + self.path
            req = urllib.request.Request(
                url, data=request_data, headers=headers, method=method
            )
            try:
                response = urllib.request.urlopen(req, timeout=12)
            except urllib.error.HTTPError as err:
                response = err
            with response:
                data = response.read(MAX_RESPONSE + 1)
                if len(data) > MAX_RESPONSE:
                    self.reply(502, b"response exceeds preview cap")
                    return
                self.send_response(response.status)
                for key in ("Content-Type", "Content-Security-Policy"):
                    v = response.headers.get(key)
                    if v:
                        self.send_header(key, v)
                self.send_header("Content-Length", str(len(data)))
                self.send_header("Cache-Control", "no-store")
                self.send_header("X-Content-Type-Options", "nosniff")
                self.end_headers()
                self.wfile.write(data)
        except (OSError, urllib.error.URLError, http.client.HTTPException):
            self.reply(502, b"preview upstream unavailable")
        finally:
            SLOTS.release()


if __name__ == "__main__":
    server = ThreadingHTTPServer(ADDRESS, Handler)
    print("RDTE-PREVIEW: public-fixture read-only proxy started on localhost:3087", flush=True)
    server.serve_forever(poll_interval=0.5)
