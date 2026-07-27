# Building a Self-Hosted Multimodal Product Recommendation System
## From Catalog Write to Ranked Search — a Working Reference Implementation

*Backend Architecture · 18 min read*

**Full source: [github.com/newmizanur/multimodal-recommendation-system](https://github.com/newmizanur/multimodal-recommendation-system)**

---

## Why self-host embeddings at all

If you've shipped a product search feature, you've probably done it the fast way: call a hosted embedding API, stuff the vectors into whatever database you already have running, and move on. That's the right call for a prototype, and it stays the right call for a long time.

There's a point, though, where the math flips. Three things usually trigger it:

- **Cost curve.** Hosted embedding APIs charge per call. That's fine at low volume, but a catalog with millions of products and continuous re-embedding on updates turns into a real line item — and it scales with your traffic, not with your infrastructure decisions.
- **Fine-tuning.** Generic multimodal encoders are trained on broad web data. They're good at "is this a shoe," mediocre at "is this the *same* shoe in a different colorway" or "does this pattern match that silhouette." You can't fine-tune a model you don't own.
- **Data residency.** Some tenants — enterprise retail clients especially — don't want product images and catalog data leaving their contracted infrastructure boundary, hosted API or not.

None of these are reasons to self-host on day one. They're reasons to know what owning the stack actually costs, so the decision is made on purpose rather than by accident six months into a cost review.

This article walks through a working architecture for a multimodal (text + image) product search and recommendation system, designed to run on Kubernetes, built around open-weight embedding models. It's opinionated where the tradeoffs are clear and honest where they aren't.

---

## Model selection: one tower or two

The first real decision is the embedding model, and it's really two decisions in one.

**Which model?** Two solid starting points:

| Model | Params | VRAM | Best for |
|---|---|---|---|
| SigLIP2 so400m | ~400M | ~3–4GB | Efficient default, good general accuracy |
| Jina-CLIP-v2 | ~865M | ~4–5GB | Better fine-tuning ceiling for fashion catalogs |

Both are unified multimodal encoders — text and images land in the same vector space, so a text query can retrieve image results and vice versa without a translation step.

**Unified encoder or two-tower?** The alternative architecture pairs a text-only encoder (BGE-M3, EmbeddingGemma) with a separate image encoder, then fuses the outputs. It's more moving parts, but it buys you something real: you can upgrade your text encoder without retraining or re-embedding your entire image catalog, and vice versa. If your text and image data evolve at different rates — which they usually do, since catalog copy changes constantly and product photography doesn't — two-tower is worth the extra operational surface. If you want the simplest thing that works, start unified with SigLIP2 and revisit later. The migration path from unified to two-tower is a re-embed, not a rewrite, so this isn't a decision you have to get perfect up front.

One detail that's easy to miss: **query/document asymmetry**. A person's search intent ("chunky white sneakers") and a catalog product photo are different kinds of input, even though the same encoder embeds both. Models like Jina-CLIP-v2 support separate prompt prefixes for query-side vs catalog-side encoding — use them. Treating a query and a catalog image as interchangeable inputs leaves accuracy on the table for free.

---

## System architecture

> See `docs/architecture.excalidraw` — open in excalidraw.com for the full diagram.

The system splits cleanly into a write path (catalog ingestion) and a read path (search/query), connected by a vector store and an embedding model server that both paths depend on. Four Go services (`catalog-writer-api`, `catalog-reader-api`, `worker`, `search-ranking-service`) plus the Python `embedding-server` live in one monorepo behind a single `go.work`, each with its own `go.mod` — a workspace, not a shared module, so each service still ships and versions independently.

### Write path: ingestion

```
Tenant Client (write)
  → Load Balancer / Ingress
  → Catalog Writer API (Go/Echo, on Kubernetes)
      ├── Write → Catalog DB (Postgres) [source of truth]
      ├── Publish → Kafka topic (product.created / product.updated / product.deleted)
      └── Presigned URL → S3-compatible storage [tenant uploads image directly]

Kafka → Worker Pool (KEDA-scaled on consumer group lag)
  ├── Read product content ← Catalog DB
  ├── Read image bytes ← S3-compatible storage
  ├── Call → Embedding Model Server (GPU inference)
  └── Write vectors → Vector DB
        └── Hard delete on product.deleted
```

Everything here except the object store is deliberately unremarkable: Postgres, Kafka, and Redis are all things you could run on any cloud, on-prem, or via whichever managed offering you prefer, without touching application code. Object storage is the one place this system leans on a specific API shape — S3's — but that API is itself implemented by more than just AWS (MinIO, Cloudflare R2, Backblaze B2), so "S3-compatible" isn't the same lock-in as "SQS" or "RDS" would be.

The load-bearing design decision here is that **the Catalog DB has exactly one writer** — the Catalog Writer API. The worker pool only ever reads from it. This eliminates a whole category of race condition where a slow embedding job overwrites a product update that landed after the job started. Postgres stays authoritative; the vector store is a derived index that can always be rebuilt from it.

Image uploads go straight from the tenant's client to S3 via a presigned URL, not through your API tier. Your API issues the URL and moves on — it never proxies image bytes.

### Read path: query

```
Tenant Client Reader
  → Load Balancer / Ingress
  → Catalog Reader API (Go/Echo)
  → Search & Ranking Service
      ├── Check Redis (query result cache) → hit: return
      ├── On miss:
      │     ├── Encode query → Embedding Model Server
      │     ├── ANN search → Vector DB (top 200–500 candidates)
      │     ├── Enrich candidates ← Redis (fallback: Catalog DB)
      │     └── Rerank (cross-encoder + business rules)
      └── Return ranked results
```

Two things worth calling out. First, the query has to go through the **same encoder** that produced the stored vectors — mixing encoder versions puts your query and your catalog in unrelated vector spaces, and cosine similarity between them is meaningless noise that happens to return *some* results, which is worse than an obvious failure. Second, ANN search returns candidates, not answers; live price and stock come from a fast enrichment lookup, not from whatever was true when the vector was written.

### Observability

All services ship telemetry through an OTel Collector into Prometheus, visualized in Grafana. One gotcha: Kafka itself doesn't emit OTel spans. Your visibility into queue behavior comes from instrumenting the producer (Catalog Writer API) and consumer (Worker) code, plus the broker's own JMX/Prometheus metrics if you're self-hosting — consumer group lag is the number that actually matters, and it's what KEDA scales on too (see the Kubernetes section below).

---

## Data model

### Catalog DB (Postgres) — source of truth

```sql
products(id, tenant_id, sku, title, description, category_id,
         price, currency, stock_qty, status, attributes jsonb, updated_at)
product_images(id, product_id, url, display_order)
categories(id, tenant_id, name, parent_id)
events(tenant_id, user_id, product_id, event_type, ts)  -- click/purchase signals
```

Multi-tenancy uses a fixed core schema plus an `attributes jsonb` column for whatever varies by vertical — size and color for fashion, wattage and voltage for electronics — with a GIN index on it. Schema-per-tenant is real overhead (migrations × N tenants); reserve it for enterprise clients with contractual isolation requirements, not as a default.

Anywhere you key on the end user, use the composite `(tenant_id, user_id)`, never `user_id` alone. It's cheap insurance against a cross-tenant data leak that only shows up in production.

### Vector DB — one row per embedding, not per product

```sql
product_embeddings(
  id             bigserial PRIMARY KEY,
  product_id     bigint    NOT NULL,
  tenant_id      text      NOT NULL,  -- ANN pre-filter, not post-filter
  embedding_type text      NOT NULL, -- 'image', 'text_title', 'text_desc', 'text_review'
  embedding      vector(768) NOT NULL,
  source_ref     text      NOT NULL, -- S3 key or text-field name this vector came from
  category_id    bigint,             -- promoted for pre-filtering
  in_stock       boolean   NOT NULL DEFAULT true, -- promoted for pre-filtering
  metadata       jsonb     NOT NULL DEFAULT '{}',
  updated_at     timestamptz NOT NULL DEFAULT now(),

  UNIQUE (product_id, embedding_type, source_ref)
)
```

A product with five photos and three text fields produces eight rows, not one. This is the natural shape for multimodal retrieval — you want to be able to match a query against any individual image or text chunk, not force everything about a product into a single averaged vector that represents nothing well.

Two operational rules matter more than the schema: **hard delete**, not soft delete, on removals — a soft-deleted vector still gets traversed during HNSW search, wasting work and silently shrinking your effective result count without any error to tell you why. And **idempotent upsert** on write, because a worker can be killed after processing a message but before committing its offset, so it will occasionally reprocess the same event. The `UNIQUE (product_id, embedding_type, source_ref)` constraint above is what makes that upsert safe — it's not decorative, it's the thing `INSERT ... ON CONFLICT (product_id, embedding_type, source_ref) DO UPDATE` keys off of. Reprocess the same Kafka message twice and you get one row overwritten twice, not two rows.

### Redis

| Cache | Key | TTL | Invalidation |
|---|---|---|---|
| Query result cache | `hash(tenant_id + query + filters)` | 30–60s | TTL only |
| Enrichment cache | `(tenant_id, product_id)` | minutes | Active write-through on price/stock change |
| Assistant session | `session_token` | ~30 min | TTL only |

The rule most teams get wrong here: **don't key the query result cache by `user_id`.** Personalization belongs at the rerank stage, applied to a shared set of retrieved candidates — not at retrieval, where per-user keys fragment your cache into thousands of near-duplicate entries and tank your hit rate for no accuracy benefit.

---

## The repository layer: no ORM, one invariant, one gotcha

Everything above is the design. This section is what shipped.

Every service follows the same shape: `delivery` (HTTP/Echo) → `usecase` (business logic) → `repository` (SQL) → Postgres, via `pgxpool`. No ORM, no query builder — every query is hand-written SQL in the repository file it belongs to. That's a deliberate constraint, not laziness: with four services touching three tables (`products`, `product_images`, `product_embeddings`) across a tenant boundary, an ORM's convenience is exactly where a missing `WHERE` clause hides.

**The one invariant that matters more than the schema:** every query in every repository — `Create`, `Get`, `Update`, `Delete`, `Search`, `Upsert`, all of it — filters by `tenant_id`. Not as a lint rule. As a literal grep gate before anything ships:

```bash
grep -A3 "func (r \*pg" catalog-writer-api/internal/repository/*.go \
  catalog-reader-api/internal/repository/*.go \
  worker/internal/repository/*.go \
  search-ranking-service/internal/repository/*.go
```

Read every match. If `tenant_id` isn't in the `WHERE` clause (or the inserted row, for `Create`/`Upsert`), that's not a style nit — it's the difference between a slow query and one tenant reading another tenant's catalog. A type system won't catch this for you; a raw-SQL repository layer means the only thing standing between tenants is someone actually reading the query.

**The pgvector gotcha:** `pgvector-go`'s `Vector` type implements `sql.Scanner`/`driver.Valuer`, which looks like it should Just Work with `pgxpool.Query`. It doesn't — pgx v5 needs the `vector` type's OID registered *per connection*, or every insert/query touching `product_embeddings` fails as soon as you actually run it. The fix is a `pgxpool.Config.AfterConnect` hook that calls `pgxvector.RegisterTypes` on every new connection the pool opens:

```go
cfg, _ := pgxpool.ParseConfig(databaseURL)
cfg.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
    return pgxvector.RegisterTypes(ctx, conn)
}
pool, _ := pgxpool.NewWithConfig(ctx, cfg)
```

Both the worker (writes vectors) and the search-ranking-service (reads them) need this — the catalog read/write services never touch `product_embeddings`, so they don't.

**Testing the vector path without a GPU.** The worker and search-ranking-service repositories only ever pass `[]float32` through to Postgres — they never look inside a vector. So validating them doesn't require the embedding model at all: seed `product_embeddings` directly with synthetic vectors, then assert on `Upsert` idempotency, `HardDeleteByProduct`, cosine ordering, and the `tenant_id`/`category_id`/`in_stock` pre-filters against a real pgvector instance. One trap worth flagging if you try this yourself: a **uniform-fill vector** (`[0.9, 0.9, 0.9, ...]`) is cosine-*identical* to any other uniform-fill vector regardless of scale — cosine similarity is angle-based, and same-value vectors always point in the same direction. Test vectors need actual variance across dimensions, or every candidate scores 1.0 and your ordering test passes for the wrong reason. Decoupling "is the SQL correct" from "is the model good" this way means the infra tests run in milliseconds, not GPU-minutes, and don't need a HuggingFace download to prove a `WHERE` clause works.

---

## ANN index strategy

HNSW is the right default — best recall/latency tradeoff for most catalog sizes, and it's what every vector DB tunes for first. IVF-PQ becomes worth the accuracy tradeoff past roughly 10M vectors, where its compression meaningfully cuts memory and cost.

Whichever index, **pre-filter by `tenant_id`** — and optionally `in_stock` or `category_id` — before or during the ANN search, not after. Post-filtering does two things badly: it risks cross-tenant leakage if a filter bug ever slips through downstream, and it wastes retrieval budget fetching candidates from other tenants you're just going to discard.

---

## Serving the embedding model

Three options, roughly in order of operational complexity:

- **FastAPI + async batching queue** — easiest to stand up, fine for moderate QPS. Start here.
- **TorchServe** — PyTorch-native, a reasonable step up.
- **Triton Inference Server** — production-grade batching and multi-model serving, NVIDIA-native. Worth it once query volume justifies the ops investment.

The non-negotiable constraint, again: query-time and ingestion-time encoding must use the exact same model version. A model upgrade is not a hot-swap — it's a re-embedding project (see below).

---

## The fine-tuning loop — the actual point of self-hosting

Everything above this section, a hosted API could technically give you. This is the part it can't.

Contrastive fine-tuning on your own product data closes the gap between "generically good at images" and "good at your catalog." Concretely:

- Train on same-item, multiple-photo pairs — different angles of the same product should embed close together.
- Mine hard negatives: visually similar but genuinely different products (near-duplicate detection working in reverse).
- Weight the attributes generic CLIP-style training underweights — pattern, material, silhouette matter enormously for fashion search and barely register in general-purpose training data.

When you ship a new model version, the embedding space itself changes — a v2 vector isn't comparable to a v1 vector even for the same image. The re-embed trigger is a first-class event: publish `product.reembed` for every affected product onto the same Kafka topic your ingestion pipeline already listens to, and let the existing worker pool process it. You don't need a separate re-indexing system; you need to enqueue work into the one you already have.

---

## Reranking: precision after recall

ANN retrieval is a recall tool — it's built to cheaply find "probably relevant" candidates across millions of vectors, not to rank them precisely. That's a second stage's job.

**Cross-encoder rerank** scores each retrieved candidate against the query jointly, rather than comparing pre-computed vectors — meaningfully more accurate, but O(N) per candidate, so it only runs on the top 50–200 ANN results, not the full candidate pool.

**Business rules** sit on top of that: stock availability, price bucket, seller trust score, and personalization signals pulled from the `events` table. This layer runs last, after the cross-encoder has done the relevance ranking — business logic adjusts an already-good ordering, it doesn't replace it.

---

## Running it on Kubernetes

```
Kubernetes cluster (EKS, GKE, AKS, or self-managed — doesn't matter which)
├── Catalog Writer API pods   — HPA on CPU/RPS
├── Catalog Reader API pods   — HPA on CPU/RPS
├── Search & Ranking pods     — HPA on CPU/RPS
├── Worker pods               — KEDA ScaledObject on Kafka consumer lag,
│                                minReplicaCount 0 (scales to zero when idle)
├── Embedding model server    — GPU, separate Deployment, own scaling
└── GPU node pool             — tainted; only Worker + embedding server tolerate it
```

Terraform wires the stateful pieces as Kubernetes-native operators rather than cloud-managed services, on purpose: **Strimzi** for Kafka, **CloudNativePG** for Postgres, Bitnami's chart for Redis. That's what makes "runs the same on EKS, GKE, AKS, or bare metal" true rather than aspirational — nothing here is an AWS API call except the S3 bucket for product images, and even that's swappable for MinIO/R2/B2 by changing `S3_ENDPOINT`, no code change.

In practice the two GPU-node-pool tolerators aren't sized the same: the worker requests 500m CPU / 512Mi memory and no GPU at all — it only ever calls the embedding server over HTTP. The embedding server is the one actually sized for the hardware it's on: 2–4 CPU, 8–16Gi memory, 1 GPU per pod, 2 replicas by default, with a 30s `initialDelaySeconds` on its readiness probe because loading a 400M-parameter model takes real time.

Two Kubernetes-specific decisions carry most of the weight here, and neither depends on which cloud (or non-cloud) the cluster runs on.

**KEDA on consumer lag for the worker pool.** Catalog bulk uploads are bursty — a tenant re-uploading their whole catalog shouldn't require capacity you're paying for 24/7. Scaling the worker pool on Kafka consumer group lag (rather than CPU) lets it scale to zero between bursts and scale out fast when one hits, decoupling ingestion throughput entirely from your steady-state API serving capacity. KEDA's `kafka` scaler trigger works against self-hosted Kafka, Redpanda, or any managed Kafka-API-compatible service — the mechanism doesn't care.

**GPU node pool isolation.** Taint the GPU nodes so CPU-only API pods can never accidentally schedule there and eat into GPU-adjacent capacity (or just waste an expensive node on a cheap workload). Only the Worker and the embedding model server carry the toleration. If you've run mixed instance-type Auto Scaling Groups on AWS before, this is the same idea as keeping GPU and CPU workloads in separate ASGs behind separate target groups — Kubernetes just expresses it as taints and tolerations, and whatever autoscaler manages your node pools (Karpenter on AWS, Cluster Autoscaler elsewhere) reacts to the same signals regardless of provider.

---

## The ops reality

This is the section that's easy to skip in an architecture writeup and expensive to skip in production.

- **Model versioning is a re-index, not a config change.** Every embedding model upgrade means the vector space shifts; there's no partial migration, only a full re-embed of the affected products.
- **GPU cost has two different shapes.** Ingestion bursts and steady-state query serving load the GPU differently — provisioning for one profile starves the other if you're not watching both.
- **Index rebuild is a real scheduling decision**, not a one-time setup step: periodic full rebuild keeps the index clean but costs a batch job window; incremental update is cheaper per-event but accumulates drift you'll eventually need to reconcile.
- **Your embedding model server's P99 latency is your query latency floor.** Everything downstream — ANN search, enrichment, rerank — adds to it; nothing downstream can hide it.
- **Breakeven is a real number, not a vibe.** GPU-hours for ingestion plus steady-state serving, compared against per-call hosted API pricing at your actual monthly query volume, tells you whether this whole exercise was worth it. Run that number before you start, not after.

---

## What's implemented versus what's designed

Worth being precise about, since this piece describes a real repo, not just a diagram:

**Built and verified against real Postgres/pgvector:** every repository method in all four Go services — catalog CRUD with soft-delete, the worker's catalog read (single query, `array_agg`-joined against `product_images`), vector `Upsert`/`HardDeleteByProduct`, and ANN `Search` with tenant/category/in-stock pre-filtering and cosine ordering. All tenant-scoped, all confirmed against a live database, not just compiled.

**Built, not yet load-bearing in this pass:** the embedding server itself (FastAPI + SigLIP2, swappable to Jina-CLIP-v2) and the full Helm/Terraform footprint for all five workloads. They run; they haven't been exercised end-to-end against a live GPU in this round.

**Deliberately stubbed:** the cross-encoder rerank client is a passthrough today — ANN score used directly as final score — pending a model decision. The Redis-miss enrichment fallback to Postgres in the search usecase is a marked TODO, not wired yet. And the write usecase publishes to Kafka *after* the Postgres write completes, with no transactional outbox between them — a crash in that gap drops the event silently. All three are known, named gaps, not surprises found later; fine for a v1, not something to carry into a flow where losing an event is expensive.

---

## Where this leaves you

None of this is exotic — it's a fairly standard event-driven ingestion pipeline (Postgres source of truth, Kafka fan-out, idempotent workers) paired with a fairly standard two-stage retrieval system (ANN recall, cross-encoder rerank), running on infrastructure patterns most backend teams already know from elsewhere. It's also, deliberately, not tied to any one cloud — Kafka, Postgres, Redis, and Kubernetes itself all run the same way regardless of provider, which matters if a client's contract dictates where things run or you just don't want a rewrite the day you switch clouds. The multimodal part is really just "the embedding model happens to accept two input types instead of one" — it doesn't change the shape of the system much.

What it *does* change is the accuracy ceiling. A hosted API gets you a good generic encoder. Owning the pipeline gets you a model that's been fine-tuned on your actual catalog, your actual near-duplicates, and your actual hard negatives — and that gap is the entire reason to take on everything in the ops section above.

---

*The core insight: none of the interesting problems here are about the model. Fine-tuning aside, this is an event-driven ingestion pipeline and a two-stage retrieval system that happen to move vectors instead of rows — the hard parts are the same ones you already know: one writer per table, idempotent consumers, pre-filter before you rank, and a tenant boundary enforced in every query because nothing else is going to enforce it for you.*

---

**Full source, migrations, Helm/Terraform, and the architecture diagram: [github.com/newmizanur/multimodal-recommendation-system](https://github.com/newmizanur/multimodal-recommendation-system)**

---

`#SystemDesign` `#GoLang` `#MachineLearning` `#VectorSearch` `#pgvector` `#Postgres` `#Kubernetes` `#Kafka` `#BackendEngineering` `#MLOps` `#RecommendationSystems`
