#!/usr/bin/env python3
import argparse
import json
import math
import os
import random
import statistics
import sys
import threading
import time
import urllib.error
import urllib.request
import uuid
from concurrent.futures import ThreadPoolExecutor
from dataclasses import dataclass
from pathlib import Path
from typing import Callable, Dict, List, Optional, Tuple

DEFAULT_PASSWORD = "PlayerPass123!"
DEFAULT_CHALLENGE_SLUG = "web-welcome"
DEFAULT_FLAG = "flag{welcome}"


@dataclass
class EndpointResult:
    name: str
    method: str
    path: str
    status: int
    latency_ms: float
    ok: bool
    error: str = ""


class ApiClient:
    def __init__(self, base_url: str, timeout: float) -> None:
        self.base_url = base_url.rstrip("/")
        self.timeout = timeout

    def request(
        self,
        method: str,
        path: str,
        body: Optional[dict] = None,
        token: Optional[str] = None,
        expect_json: bool = True,
    ) -> Tuple[int, object, Dict[str, str], bytes]:
        url = self.base_url + path
        data = None
        headers = {"Accept": "application/json"}
        if body is not None:
            data = json.dumps(body, separators=(",", ":")).encode("utf-8")
            headers["Content-Type"] = "application/json"
        if token:
            headers["Authorization"] = f"Bearer {token}"
        request = urllib.request.Request(url, data=data, headers=headers, method=method)
        try:
            with urllib.request.urlopen(request, timeout=self.timeout) as response:
                raw = response.read()
                status = response.getcode()
                response_headers = dict(response.headers.items())
        except urllib.error.HTTPError as exc:
            raw = exc.read()
            status = exc.code
            response_headers = dict(exc.headers.items())
        if not expect_json:
            return status, raw, response_headers, raw
        if not raw:
            return status, {}, response_headers, raw
        try:
            payload = json.loads(raw.decode("utf-8"))
        except json.JSONDecodeError:
            payload = {"raw": raw.decode("utf-8", errors="replace")}
        return status, payload, response_headers, raw


class MetricsSnapshot:
    def __init__(self, raw_text: str, values: Dict[str, float]) -> None:
        self.raw_text = raw_text
        self.values = values

    @classmethod
    def from_text(cls, text: str) -> "MetricsSnapshot":
        values: Dict[str, float] = {}
        for line in text.splitlines():
            line = line.strip()
            if not line or line.startswith("#"):
                continue
            parts = line.split(None, 1)
            if len(parts) != 2:
                continue
            name, value = parts
            try:
                values[name] = float(value)
            except ValueError:
                continue
        return cls(text, values)

    def delta(self, other: "MetricsSnapshot") -> Dict[str, float]:
        keys = set(self.values) | set(other.values)
        diff: Dict[str, float] = {}
        for key in keys:
            value = other.values.get(key, 0.0) - self.values.get(key, 0.0)
            if abs(value) > 1e-9:
                diff[key] = value
        return diff


class LoadContext:
    def __init__(self, args: argparse.Namespace) -> None:
        self.args = args
        self.client = ApiClient(args.base_url, args.timeout_seconds)
        self.challenge_id: Optional[int] = None
        self.player_email: Optional[str] = None
        self.player_token: Optional[str] = None
        self.player_username: Optional[str] = None
        self.random = random.Random(args.seed)
        self.random_lock = threading.Lock()

    def rand_choice(self, items: List[Tuple[str, Callable[[], EndpointResult], int]]) -> Tuple[str, Callable[[], EndpointResult], int]:
        with self.random_lock:
            total = sum(weight for _, _, weight in items)
            target = self.random.randint(1, total)
            seen = 0
            for item in items:
                seen += item[2]
                if target <= seen:
                    return item
        return items[-1]


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Baseline load test runner for the CTF API")
    parser.add_argument("--base-url", default=os.environ.get("BASE_URL", "http://127.0.0.1:8080"))
    parser.add_argument("--scenario", choices=["public", "player", "submit", "login", "instance"], default="player")
    parser.add_argument("--concurrency", type=int, default=8)
    parser.add_argument("--duration-seconds", type=int, default=30)
    parser.add_argument("--timeout-seconds", type=float, default=5.0)
    parser.add_argument("--seed", type=int, default=42)
    parser.add_argument("--challenge-slug", default=DEFAULT_CHALLENGE_SLUG)
    parser.add_argument("--flag", default=DEFAULT_FLAG)
    parser.add_argument("--player-email")
    parser.add_argument("--player-password", default=os.environ.get("PLAYER_PASSWORD", DEFAULT_PASSWORD))
    parser.add_argument("--player-token")
    parser.add_argument("--register-player", action="store_true")
    parser.add_argument("--register-prefix", default="loadtest")
    parser.add_argument("--metrics", action="store_true", default=True)
    parser.add_argument("--no-metrics", dest="metrics", action="store_false")
    parser.add_argument("--output-dir", default="")
    parser.add_argument("--report-json", default="")
    parser.add_argument("--max-error-rate", type=float, default=-1.0)
    parser.add_argument("--max-p95-ms", type=float, default=-1.0)
    return parser.parse_args()


def ensure_directory(path: str) -> Path:
    target = Path(path)
    target.mkdir(parents=True, exist_ok=True)
    return target


def fetch_metrics(ctx: LoadContext) -> MetricsSnapshot:
    status, _, _, raw = ctx.client.request("GET", "/api/v1/metrics", expect_json=False)
    if status != 200:
        raise RuntimeError(f"metrics endpoint returned HTTP {status}")
    return MetricsSnapshot.from_text(raw.decode("utf-8", errors="replace"))


def resolve_challenge_id(ctx: LoadContext) -> int:
    status, payload, _, _ = ctx.client.request("GET", "/api/v1/challenges")
    if status != 200:
        raise RuntimeError(f"failed to list challenges: HTTP {status}")
    items = payload.get("items", []) if isinstance(payload, dict) else []
    for item in items:
        if item.get("slug") == ctx.args.challenge_slug:
            challenge_id = item.get("id")
            if challenge_id is None:
                break
            return int(challenge_id)
    raise RuntimeError(f"challenge slug {ctx.args.challenge_slug!r} not found")


def ensure_player(ctx: LoadContext) -> None:
    if ctx.args.player_token:
        ctx.player_token = ctx.args.player_token
        ctx.player_email = ctx.args.player_email or "token@provided.invalid"
        return

    email = ctx.args.player_email
    password = ctx.args.player_password
    if not email and not ctx.args.register_player:
        raise RuntimeError("set --player-email or use --register-player")

    if ctx.args.register_player:
        unique = uuid.uuid4().hex[:10]
        username = f"{ctx.args.register_prefix}_{unique}"
        email = f"{username}@example.com"
        display_name = f"Load Test {unique}"
        status, payload, _, _ = ctx.client.request(
            "POST",
            "/api/v1/auth/register",
            body={
                "username": username,
                "email": email,
                "password": password,
                "display_name": display_name,
            },
        )
        if status != 201:
            raise RuntimeError(f"failed to register load-test player: HTTP {status}, response={payload}")
        ctx.player_username = username
        ctx.player_email = email
    else:
        ctx.player_email = email

    status, payload, _, _ = ctx.client.request(
        "POST",
        "/api/v1/auth/login",
        body={"identifier": ctx.player_email, "password": password},
    )
    if status != 200:
        raise RuntimeError(f"failed to login load-test player: HTTP {status}, response={payload}")
    token = payload.get("token") if isinstance(payload, dict) else None
    if not token:
        raise RuntimeError("login response did not include token")
    ctx.player_token = str(token)
    if not ctx.player_username and isinstance(payload, dict):
        user = payload.get("user") or {}
        username = user.get("username")
        if username:
            ctx.player_username = str(username)


def call_endpoint(
    ctx: LoadContext,
    name: str,
    method: str,
    path: str,
    body: Optional[dict] = None,
    token: Optional[str] = None,
    expected_statuses: Tuple[int, ...] = (200,),
) -> EndpointResult:
    started = time.perf_counter()
    try:
        status, payload, _, _ = ctx.client.request(method, path, body=body, token=token)
        latency_ms = (time.perf_counter() - started) * 1000.0
        ok = status in expected_statuses
        error = ""
        if not ok:
            if isinstance(payload, dict):
                code = payload.get("error")
                message = payload.get("message")
                error = f"http_{status}:{code or message or 'unexpected_response'}"
            else:
                error = f"http_{status}:unexpected_response"
        return EndpointResult(name=name, method=method, path=path, status=status, latency_ms=latency_ms, ok=ok, error=error)
    except Exception as exc:  # pragma: no cover - runtime/network failures are environment-driven
        latency_ms = (time.perf_counter() - started) * 1000.0
        return EndpointResult(name=name, method=method, path=path, status=0, latency_ms=latency_ms, ok=False, error=str(exc))


def public_steps(ctx: LoadContext) -> List[Tuple[str, Callable[[], EndpointResult], int]]:
    challenge_path = f"/api/v1/challenges/{ctx.challenge_id}"
    return [
        ("announcements", lambda: call_endpoint(ctx, "announcements", "GET", "/api/v1/announcements"), 1),
        ("challenges", lambda: call_endpoint(ctx, "challenges", "GET", "/api/v1/challenges"), 3),
        ("challenge_detail", lambda: call_endpoint(ctx, "challenge_detail", "GET", challenge_path), 3),
        ("scoreboard", lambda: call_endpoint(ctx, "scoreboard", "GET", "/api/v1/scoreboard"), 2),
    ]


def player_steps(ctx: LoadContext) -> List[Tuple[str, Callable[[], EndpointResult], int]]:
    assert ctx.player_token is not None
    steps = list(public_steps(ctx))
    steps.extend(
        [
            ("me", lambda: call_endpoint(ctx, "me", "GET", "/api/v1/me", token=ctx.player_token), 2),
            ("me_submissions", lambda: call_endpoint(ctx, "me_submissions", "GET", "/api/v1/me/submissions", token=ctx.player_token), 1),
            ("me_solves", lambda: call_endpoint(ctx, "me_solves", "GET", "/api/v1/me/solves", token=ctx.player_token), 1),
        ]
    )
    return steps


def submit_steps(ctx: LoadContext) -> List[Tuple[str, Callable[[], EndpointResult], int]]:
    assert ctx.player_token is not None
    steps = list(player_steps(ctx))
    submit_path = f"/api/v1/challenges/{ctx.challenge_id}/submissions"
    steps.append(
        (
            "submit_flag",
            lambda: call_endpoint(
                ctx,
                "submit_flag",
                "POST",
                submit_path,
                body={"flag": ctx.args.flag},
                token=ctx.player_token,
                expected_statuses=(200, 429),
            ),
            1,
        )
    )
    return steps


def login_steps(ctx: LoadContext) -> List[Tuple[str, Callable[[], EndpointResult], int]]:
    if not ctx.player_email:
        raise RuntimeError("login scenario requires --player-email or --register-player")
    payload = {"identifier": ctx.player_email, "password": ctx.args.player_password}
    return [
        ("login", lambda: call_endpoint(ctx, "login", "POST", "/api/v1/auth/login", body=payload, expected_statuses=(200, 429)), 1),
    ]


def ensure_instance_ready(ctx: LoadContext) -> None:
    assert ctx.player_token is not None
    path = f"/api/v1/challenges/{ctx.challenge_id}/instances/me"
    result = call_endpoint(ctx, "instance_create_warmup", "POST", path, token=ctx.player_token, expected_statuses=(200, 201))
    if not result.ok:
        raise RuntimeError(f"failed to prepare dynamic instance: status={result.status} error={result.error}")


def instance_steps(ctx: LoadContext) -> List[Tuple[str, Callable[[], EndpointResult], int]]:
    assert ctx.player_token is not None
    instance_path = f"/api/v1/challenges/{ctx.challenge_id}/instances/me"
    renew_path = instance_path + "/renew"
    return [
        (
            "instance_create",
            lambda: call_endpoint(ctx, "instance_create", "POST", instance_path, token=ctx.player_token, expected_statuses=(200, 201, 409)),
            2,
        ),
        (
            "instance_get",
            lambda: call_endpoint(ctx, "instance_get", "GET", instance_path, token=ctx.player_token, expected_statuses=(200, 404)),
            4,
        ),
        (
            "instance_renew",
            lambda: call_endpoint(ctx, "instance_renew", "POST", renew_path, token=ctx.player_token, expected_statuses=(200, 404, 409)),
            1,
        ),
        (
            "instance_delete",
            lambda: call_endpoint(ctx, "instance_delete", "DELETE", instance_path, token=ctx.player_token, expected_statuses=(200, 404)),
            1,
        ),
    ]


def cleanup_instance(ctx: LoadContext) -> None:
    if ctx.player_token is None or ctx.challenge_id is None:
        return
    path = f"/api/v1/challenges/{ctx.challenge_id}/instances/me"
    _ = call_endpoint(ctx, "instance_cleanup", "DELETE", path, token=ctx.player_token, expected_statuses=(200, 404))


def build_steps(ctx: LoadContext) -> List[Tuple[str, Callable[[], EndpointResult], int]]:
    if ctx.args.scenario == "public":
        return public_steps(ctx)
    if ctx.args.scenario == "player":
        return player_steps(ctx)
    if ctx.args.scenario == "submit":
        return submit_steps(ctx)
    if ctx.args.scenario == "login":
        return login_steps(ctx)
    if ctx.args.scenario == "instance":
        ensure_instance_ready(ctx)
        return instance_steps(ctx)
    raise RuntimeError(f"unsupported scenario: {ctx.args.scenario}")


def percentile(sorted_values: List[float], pct: float) -> float:
    if not sorted_values:
        return 0.0
    if len(sorted_values) == 1:
        return sorted_values[0]
    rank = pct * (len(sorted_values) - 1)
    lower = math.floor(rank)
    upper = math.ceil(rank)
    if lower == upper:
        return sorted_values[lower]
    lower_value = sorted_values[lower]
    upper_value = sorted_values[upper]
    return lower_value + (upper_value - lower_value) * (rank - lower)


def run_load(ctx: LoadContext, steps: List[Tuple[str, Callable[[], EndpointResult], int]]) -> List[EndpointResult]:
    deadline = time.perf_counter() + ctx.args.duration_seconds
    results: List[EndpointResult] = []
    results_lock = threading.Lock()

    def worker() -> None:
        local_results: List[EndpointResult] = []
        while time.perf_counter() < deadline:
            _, step, _ = ctx.rand_choice(steps)
            local_results.append(step())
        with results_lock:
            results.extend(local_results)

    with ThreadPoolExecutor(max_workers=ctx.args.concurrency) as executor:
        futures = [executor.submit(worker) for _ in range(ctx.args.concurrency)]
        for future in futures:
            future.result()
    return results


def summarize(results: List[EndpointResult]) -> Dict[str, object]:
    latencies = sorted(item.latency_ms for item in results)
    total = len(results)
    ok_count = sum(1 for item in results if item.ok)
    error_count = total - ok_count
    total_duration_ms = sum(item.latency_ms for item in results)
    by_name: Dict[str, Dict[str, object]] = {}
    errors: Dict[str, int] = {}
    for item in results:
        bucket = by_name.setdefault(
            item.name,
            {
                "method": item.method,
                "path": item.path,
                "total": 0,
                "ok": 0,
                "error": 0,
                "latencies": [],
                "statuses": {},
            },
        )
        bucket["total"] = int(bucket["total"]) + 1
        bucket["ok"] = int(bucket["ok"]) + (1 if item.ok else 0)
        bucket["error"] = int(bucket["error"]) + (0 if item.ok else 1)
        bucket["latencies"].append(item.latency_ms)
        statuses = bucket["statuses"]
        statuses[str(item.status)] = int(statuses.get(str(item.status), 0)) + 1
        if item.error:
            errors[item.error] = errors.get(item.error, 0) + 1

    endpoint_summary: Dict[str, Dict[str, object]] = {}
    for name, bucket in sorted(by_name.items()):
        endpoint_latencies = sorted(bucket.pop("latencies"))
        endpoint_total = int(bucket["total"])
        endpoint_errors = int(bucket["error"])
        endpoint_summary[name] = {
            "method": bucket["method"],
            "path": bucket["path"],
            "total": endpoint_total,
            "ok": int(bucket["ok"]),
            "error": endpoint_errors,
            "error_rate": endpoint_errors / endpoint_total if endpoint_total else 0.0,
            "avg_ms": (sum(endpoint_latencies) / endpoint_total) if endpoint_total else 0.0,
            "p95_ms": percentile(endpoint_latencies, 0.95),
            "max_ms": endpoint_latencies[-1] if endpoint_latencies else 0.0,
            "statuses": bucket["statuses"],
        }

    return {
        "total_requests": total,
        "ok_requests": ok_count,
        "error_requests": error_count,
        "error_rate": (error_count / total) if total else 0.0,
        "avg_ms": (sum(latencies) / total) if total else 0.0,
        "median_ms": statistics.median(latencies) if latencies else 0.0,
        "p95_ms": percentile(latencies, 0.95),
        "max_ms": latencies[-1] if latencies else 0.0,
        "approx_sequential_time_ms": total_duration_ms,
        "errors": dict(sorted(errors.items(), key=lambda item: (-item[1], item[0]))),
        "endpoints": endpoint_summary,
    }


def print_summary(args: argparse.Namespace, summary: Dict[str, object], elapsed_seconds: float, metrics_delta: Optional[Dict[str, float]]) -> None:
    total_requests = int(summary["total_requests"])
    ok_requests = int(summary["ok_requests"])
    error_requests = int(summary["error_requests"])
    error_rate = float(summary["error_rate"])
    avg_ms = float(summary["avg_ms"])
    p95_ms = float(summary["p95_ms"])
    max_ms = float(summary["max_ms"])
    rps = total_requests / elapsed_seconds if elapsed_seconds > 0 else 0.0

    print(f"scenario: {args.scenario}")
    print(f"duration_seconds: {elapsed_seconds:.2f}")
    print(f"concurrency: {args.concurrency}")
    print(f"total_requests: {total_requests}")
    print(f"requests_per_second: {rps:.2f}")
    print(f"ok_requests: {ok_requests}")
    print(f"error_requests: {error_requests}")
    print(f"error_rate: {error_rate:.4f}")
    print(f"avg_ms: {avg_ms:.2f}")
    print(f"p95_ms: {p95_ms:.2f}")
    print(f"max_ms: {max_ms:.2f}")
    print("endpoints:")
    endpoints = summary["endpoints"]
    for name, bucket in endpoints.items():
        print(
            "  - {name}: total={total} ok={ok} error={error} avg_ms={avg:.2f} p95_ms={p95:.2f} max_ms={max_ms:.2f}".format(
                name=name,
                total=int(bucket["total"]),
                ok=int(bucket["ok"]),
                error=int(bucket["error"]),
                avg=float(bucket["avg_ms"]),
                p95=float(bucket["p95_ms"]),
                max_ms=float(bucket["max_ms"]),
            )
        )
    errors = summary["errors"]
    if errors:
        print("errors:")
        for key, count in errors.items():
            print(f"  - {key}: {count}")
    if metrics_delta:
        print("metrics_delta:")
        interesting = sorted(metrics_delta.items(), key=lambda item: (-abs(item[1]), item[0]))
        for key, value in interesting[:20]:
            print(f"  - {key}: {value:g}")


def maybe_write_outputs(
    args: argparse.Namespace,
    pre_metrics: Optional[MetricsSnapshot],
    post_metrics: Optional[MetricsSnapshot],
    summary: Dict[str, object],
    elapsed_seconds: float,
) -> Optional[Dict[str, float]]:
    metrics_delta = None
    if pre_metrics and post_metrics:
        metrics_delta = pre_metrics.delta(post_metrics)

    report = {
        "generated_at": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
        "base_url": args.base_url,
        "scenario": args.scenario,
        "concurrency": args.concurrency,
        "duration_seconds": args.duration_seconds,
        "elapsed_seconds": elapsed_seconds,
        "challenge_slug": args.challenge_slug,
        "metrics_enabled": args.metrics,
        "summary": summary,
        "metrics_delta": metrics_delta or {},
    }

    if args.output_dir:
        output_dir = ensure_directory(args.output_dir)
        (output_dir / "report.json").write_text(json.dumps(report, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
        if pre_metrics:
            (output_dir / "metrics.before.txt").write_text(pre_metrics.raw_text, encoding="utf-8")
        if post_metrics:
            (output_dir / "metrics.after.txt").write_text(post_metrics.raw_text, encoding="utf-8")

    if args.report_json:
        report_path = Path(args.report_json)
        if report_path.parent != Path(""):
            report_path.parent.mkdir(parents=True, exist_ok=True)
        report_path.write_text(json.dumps(report, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")

    return metrics_delta


def enforce_thresholds(args: argparse.Namespace, summary: Dict[str, object]) -> int:
    failures: List[str] = []
    if args.max_error_rate >= 0 and float(summary["error_rate"]) > args.max_error_rate:
        failures.append(f"error_rate {float(summary['error_rate']):.4f} exceeded threshold {args.max_error_rate:.4f}")
    if args.max_p95_ms >= 0 and float(summary["p95_ms"]) > args.max_p95_ms:
        failures.append(f"p95_ms {float(summary['p95_ms']):.2f} exceeded threshold {args.max_p95_ms:.2f}")
    if failures:
        for failure in failures:
            print(f"threshold_failed: {failure}", file=sys.stderr)
        return 1
    return 0


def main() -> int:
    args = parse_args()
    ctx = LoadContext(args)

    ctx.challenge_id = resolve_challenge_id(ctx)
    needs_player = args.scenario in {"player", "submit", "login", "instance"}
    if needs_player:
        ensure_player(ctx)

    pre_metrics = fetch_metrics(ctx) if args.metrics else None
    steps = build_steps(ctx)

    started = time.perf_counter()
    try:
        results = run_load(ctx, steps)
    finally:
        if args.scenario == "instance":
            cleanup_instance(ctx)
    elapsed_seconds = time.perf_counter() - started
    post_metrics = fetch_metrics(ctx) if args.metrics else None

    summary = summarize(results)
    metrics_delta = maybe_write_outputs(args, pre_metrics, post_metrics, summary, elapsed_seconds)
    print_summary(args, summary, elapsed_seconds, metrics_delta)
    return enforce_thresholds(args, summary)


if __name__ == "__main__":
    sys.exit(main())
