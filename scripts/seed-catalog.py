#!/usr/bin/env python3
"""
Seeds a tenant's catalog with sample products via the Catalog Writer API,
for local dev / load testing the ingestion pipeline (Kafka -> worker ->
embedding server -> vector DB) end to end.

Usage:
    python scripts/seed-catalog.py --tenant acme-marketplace --count 500
"""
import argparse
import random
import uuid
import requests

ADJECTIVES = ["Classic", "Vintage", "Modern", "Slim", "Relaxed", "Cropped"]
NOUNS = ["Sneaker", "Jacket", "Tote Bag", "T-Shirt", "Watch", "Sunglasses"]
# category_id is a FK into the categories table (see
# 00001_create_table_categories.sql) — these must already exist for the
# tenant being seeded, e.g.:
#   INSERT INTO categories (tenant_id, name) VALUES
#     ('acme-marketplace','footwear'), ('acme-marketplace','outerwear'),
#     ('acme-marketplace','accessories'), ('acme-marketplace','apparel');
CATEGORY_IDS = ["1", "2", "3", "4"]


def random_product():
    return {
        "sku": f"SKU-{uuid.uuid4().hex[:10].upper()}",
        "title": f"{random.choice(ADJECTIVES)} {random.choice(NOUNS)}",
        "description": "Sample product generated for local ingestion testing.",
        "category_id": random.choice(CATEGORY_IDS),
        "price": round(random.uniform(15, 250), 2),
        "currency": "USD",
        "stock_qty": random.randint(0, 200),
        "attributes": {"color": random.choice(["black", "white", "navy", "red"])},
    }


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--tenant", default="acme-marketplace")
    parser.add_argument("--api-key", default="acme-marketplace-dev-key", help="X-API-Key for the tenant")
    parser.add_argument("--count", type=int, default=100)
    parser.add_argument("--api-base", default="http://localhost:8080")
    args = parser.parse_args()

    url = f"{args.api_base}/v1/tenants/{args.tenant}/products"
    headers = {"X-API-Key": args.api_key}
    created = 0
    for _ in range(args.count):
        resp = requests.post(url, json=random_product(), headers=headers, timeout=10)
        if resp.ok:
            created += 1
        else:
            print(f"failed: {resp.status_code} {resp.text}")

    print(f"seeded {created}/{args.count} products for tenant={args.tenant}")


if __name__ == "__main__":
    main()
