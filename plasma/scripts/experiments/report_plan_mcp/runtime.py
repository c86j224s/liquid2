"""Subprocess and loopback HTTP primitives for isolated product runs."""

from __future__ import annotations

import json
import os
from pathlib import Path
import subprocess
import sys
import time
from typing import Any, Mapping
from urllib import error, request

from .models import RunManifest


class ProductRunError(RuntimeError):
    def __init__(self, message: str, *, started: bool = False, kind: str = "runtime") -> None:
        super().__init__(message)
        self.started = started
        self.kind = kind
        self.manifest: RunManifest | None = None


def start_connector_stub(port: int, environment: Mapping[str, str], log: object) -> subprocess.Popen[bytes]:
    script = """
import http.server, json, sys
class Handler(http.server.BaseHTTPRequestHandler):
    def do_GET(self):
        body = json.dumps({"status": "isolated", "connector": "disabled"}).encode()
        self.send_response(200 if self.path in ("/", "/health", "/api/health") else 503)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers(); self.wfile.write(body)
    def log_message(self, fmt, *args):
        print(fmt % args, flush=True)
http.server.ThreadingHTTPServer(("127.0.0.1", int(sys.argv[1])), Handler).serve_forever()
"""
    return subprocess.Popen([sys.executable, "-u", "-c", script, str(port)], env=environment, stdout=log, stderr=subprocess.STDOUT)


def wait_health(base: str, process: subprocess.Popen[bytes], timeout: int, *, started: bool = False) -> None:
    deadline = time.monotonic() + timeout
    while time.monotonic() < deadline:
        if process.poll() is not None:
            raise ProductRunError("serve exited during health check", started=started, kind="health")
        try:
            if http_json(f"{base}/api/health"):
                return
        except (error.URLError, ProductRunError):
            time.sleep(0.1)
    raise ProductRunError("serve health check timed out", started=started, kind="health")


def http_json(url: str, body: Mapping[str, object] | None = None) -> dict[str, Any]:
    data = None if body is None else json.dumps(body).encode()
    headers = {} if data is None else {"Content-Type": "application/json"}
    with request.urlopen(request.Request(url, data=data, headers=headers), timeout=10) as response:
        value = json.load(response)
    if not isinstance(value, dict):
        raise ProductRunError("HTTP response is not an object")
    return value


def post_json(url: str, body: Mapping[str, object], *, started: bool) -> dict[str, Any]:
    try:
        return http_json(url, body)
    except (error.URLError, ProductRunError) as exc:
        raise ProductRunError("report start request failed", started=started) from exc


def http_bytes(url: str) -> bytes:
    with request.urlopen(url, timeout=30) as response:
        return response.read()


def run_json(command: list[str], environment: Mapping[str, str], *, started: bool) -> dict[str, Any]:
    try:
        completed = subprocess.run(command, env=environment, check=True, capture_output=True, text=True)
        value = json.loads(completed.stdout)
    except (subprocess.CalledProcessError, json.JSONDecodeError) as exc:
        raise ProductRunError("product CLI failed", started=started) from exc
    if not isinstance(value, dict):
        raise ProductRunError("CLI response is not an object", started=started)
    return value


def find_string(value: object, *keys: str) -> str:
    if isinstance(value, dict):
        for key in keys:
            found = value.get(key)
            if isinstance(found, str) and found:
                return found
        for nested in value.values():
            try:
                return find_string(nested, *keys)
            except ProductRunError:
                pass
    raise ProductRunError(f"CLI response omitted {keys}")


def child_environment(overrides: Mapping[str, str | None]) -> dict[str, str]:
    environment = dict(os.environ)
    for key, value in overrides.items():
        if value is None:
            environment.pop(key, None)
        else:
            environment[key] = value
            Path(value).mkdir(parents=True, exist_ok=True)
    return environment


def stop_process(process: subprocess.Popen[bytes]) -> None:
    if process.poll() is not None:
        return
    process.terminate()
    try:
        process.wait(timeout=5)
    except subprocess.TimeoutExpired:
        process.kill()
        process.wait(timeout=5)
