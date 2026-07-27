#!/usr/bin/env python3
"""
Benchmarks ANN retrieval latency/recall against the Search & Ranking
Service — useful for the HNSW vs IVF-PQ decision at your actual catalog
size (see article: HNSW default, IVF-PQ past ~10M vectors).

Usage:
    python scripts/benchmark-ann.py --tenant acme-marketplace --queries queries.txt --top-k 200
"""
import argparse
import statistics
import time

import requests


def load_queries(path):
    with open(path) as f:
        return [line.strip() for line in f if line.strip()]


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--tenant", default="acme-marketplace")
    parser.add_argument("--api-key", default="acme-marketplace-dev-key", help="X-API-Key for the tenant")
    parser.add_argument("--queries", required=True, help="text file, one query per line")
    parser.add_argument("--top-k", type=int, default=200)
    parser.add_argument("--api-base", default="http://localhost:8081")
    args = parser.parse_args()

    queries = load_queries(args.queries)
    headers = {"X-API-Key": args.api_key}
    latencies_ms = []

    for q in queries:
        start = time.perf_counter()
        resp = requests.post(
            f"{args.api_base}/v1/tenants/{args.tenant}/search",
            json={"query": q, "filters": {}},
            headers=headers,
            timeout=10,
        )
        elapsed_ms = (time.perf_counter() - start) * 1000
        latencies_ms.append(elapsed_ms)
        status = "ok" if resp.ok else f"HTTP {resp.status_code}"
        print(f"{elapsed_ms:7.1f}ms  {status}  {q!r}")

    if latencies_ms:
        print("\n--- summary ---")
        print(f"count:  {len(latencies_ms)}")
        print(f"p50:    {statistics.median(latencies_ms):.1f}ms")
        latencies_ms.sort()
        p95 = latencies_ms[int(len(latencies_ms) * 0.95)]
        print(f"p95:    {p95:.1f}ms")
        print(f"max:    {max(latencies_ms):.1f}ms")


if __name__ == "__main__":
    main()
