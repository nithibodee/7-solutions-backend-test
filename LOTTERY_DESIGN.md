# Lottery Ticket Search System — Design Proposal

> Design exercise only. No implementation. Section 2 of the coding test.
>
> A presentation-oriented walkthrough of this same design lives at
> [`docs/lottery-brief.html`](docs/lottery-brief.html) — open it in a browser.

## 1. Problem statement

- **Dataset:** 10,000,000 lottery tickets. Each ticket carries a **6-digit
  number** (`000000`–`999999`).
- **Search:** a 6-character pattern of digits and `*` wildcards
  (`****23`, `1****5`, `123***`, `1*3*5*`, …).
- **Distribution constraint:** the *same pattern* must **not hand the same
  ticket to two users at the same time**. Matching tickets are allocated to
  requesting users without duplicate simultaneous selection.
- **Performance:** search + allocation must stay fast at 10M+ rows.

### 1.1 Scale analysis (drives every later choice)

| Quantity | Value | Consequence |
|---|---|---|
| Possible 6-digit values | 10⁶ = 1,000,000 | with 10M tickets, ~10 tickets share each number |
| Fixed digits in a pattern, `k` | 0–6 | |
| Expected matches for `k` fixed digits | 10M / 10ᵏ | `k=2` → 100k, `k=3` → 10k, `k=1` → 1M, `k=0` → everything |
| Distinct possible patterns | 11⁶ ≈ 1.77M | too many to fully precompute and keep fresh on writes |
| Raw data size | 10M × (~40 B) ≈ 400 MB | **fits comfortably in RAM and in a single Postgres instance** |

**Takeaways:**
1. The data is *small*. This is not a big-data problem; it is a **concurrency
   and allocation** problem.
2. Result sets for loose patterns (`k ≤ 1`) are huge — the API must cap /
   paginate them and allocation should require `k ≥ 2`.
3. Full precomputation of all patterns is not worth the write-time cost; compute
   on demand and cache hot patterns.

---

## 2. Recommended architecture

```
                 ┌─────────────────────────────────────────────┐
                 │                  API layer                   │
                 │   POST /search   { pattern, count, userId }  │
                 └───────────────┬─────────────────────────────┘
                                 │
                 ┌───────────────▼───────────────┐
                 │      Search / Allocation       │
                 │            service             │
                 └───────┬───────────────┬───────┘
             candidate IDs│               │authoritative reserve
       ┌──────────────────▼────┐   ┌──────▼──────────────────────────┐
       │  In-memory index       │   │        PostgreSQL               │
       │  (Roaring bitmaps,      │   │  tickets(id, number, d1..d6,    │
       │   60 posting lists)     │   │          status, reserved_*)    │
       │  optional, for hot QPS  │   │  + reservations(ticket_id …)    │
       │  rebuilt from PG        │   │  SELECT … FOR UPDATE SKIP LOCKED │
       └────────────────────────┘   └─────────────┬───────────────────┘
                                                  │ lease expiry / confirm
                                          ┌───────▼────────┐
                                          │  Reaper job     │
                                          │ (reclaim leases)│
                                          └────────────────┘
```

**System of record + allocation:** PostgreSQL.
**Optional acceleration:** an in-process Roaring-bitmap index for candidate
generation when search QPS is high. It is *only* a filter — every ticket handed
to a user is still reserved transactionally in Postgres, so the index may lag
without ever causing a double-allocation.

---

## 3. Storage / database choice

### 3.1 Recommendation: **PostgreSQL** (single primary, optionally + read replicas)

| Reason | Detail |
|---|---|
| **The allocation primitive is built in** | `SELECT … FOR UPDATE SKIP LOCKED` lets N concurrent requests for the same pattern each grab a *disjoint* set of rows with no blocking and no application-level locking. This is exactly the "competing consumers on a queue" pattern and it solves the core constraint essentially for free. |
| **ACID reservations** | A reservation is a row state transition inside a transaction. Crash/rollback can't leave a ticket half-allocated. |
| **The data is tiny** | 10M narrow rows (~0.5–1 GB with indexes). Everything stays in `shared_buffers` / OS cache; no sharding needed. |
| **Flexible indexing** | B-tree (prefix), expression index on `reverse()` (suffix), per-digit columns with automatic **BitmapAnd** (scattered wildcards), `pg_trgm` GIN (arbitrary substring) — all in one engine. |
| **Operational simplicity** | One well-understood system for storage, search, and allocation. Backups, HA (Patroni), and monitoring are solved problems. |
| **Correct under horizontal API scale-out** | All API instances share one Postgres; `SKIP LOCKED` coordinates across connections, so adding API pods needs zero extra coordination service. |

### 3.2 Alternatives considered

| Option | Strengths | Why not primary |
|---|---|---|
| **Redis** | `SET NX EX` gives atomic per-ticket lease; in-memory speed; native TTL for lease expiry. | Not a system of record (durability/replication caveats). Still need a durable store behind it. **Used as a complement**, not the SoR — see §5.3. |
| **Elasticsearch / OpenSearch** | First-class `wildcard` and `regexp` queries; scales horizontally. | Near-real-time (not transactional); **no row-lock / SKIP-LOCKED equivalent**, so the distribution constraint would still need Postgres or Redis. Operationally heavier. Overkill for 10M fixed-format rows. |
| **MongoDB** | Easy ops; `findAndModify` is atomic per document. | Per-document atomicity helps, but there is no `SKIP LOCKED`: concurrent claimers of the same pattern collide on the same "first" documents and must retry, wasting work. Regex/wildcard queries are not well indexed unless anchored (prefix). |
| **Pure in-memory service (no DB)** | Fastest possible search. | Loses durability and multi-node consistency; rebuilding allocation state after a crash is error-prone. Fine as a *cache/index*, not as the SoR. |
| **Precompute every pattern → KV store** | O(1) lookup. | 1.77M pattern keys, each needing maintenance on every ticket insert/void; most keys are never read. Poor cost/benefit vs. on-demand + hot-pattern cache. |

---

## 4. Wildcard matching — algorithm & indexing

### 4.1 Schema (denormalised for the planner)

```
tickets(
  id            bigint PRIMARY KEY,
  number        char(6)  NOT NULL,          -- '004523'
  d1..d6        smallint NOT NULL,          -- individual digits, 0–9
  number_rev    char(6)  GENERATED ALWAYS AS (reverse(number)) STORED,
  status        text     NOT NULL DEFAULT 'available',  -- available | reserved | sold | void
  reserved_by   uuid,
  reserved_at   timestamptz,
  reserved_until timestamptz
)
```

Indexes:

| Index | Serves |
|---|---|
| `btree(number text_pattern_ops)` | **prefix** patterns `123***` → `number LIKE '123%'` (range scan) |
| `btree(number_rev text_pattern_ops)` | **suffix** patterns `***456` → `number_rev LIKE '654%'` |
| `btree(d1) … btree(d6)` (six single-column) | **scattered** patterns `1**4*6` → planner does `BitmapAnd` of the fixed-digit indexes |
| `btree(status, reserved_until)` partial `WHERE status IN ('available','reserved')` | fast "is this ticket claimable right now" filter during allocation |
| `pg_trgm GIN(number)` *(optional)* | fully arbitrary `LIKE '%..%'` / regex, fallback for odd patterns |

### 4.2 Query strategy by pattern class

| Pattern class | Example | Plan |
|---|---|---|
| Prefix only | `123***` | B-tree range scan on `number` |
| Suffix only | `***456` | B-tree range scan on `number_rev` |
| Prefix **and** suffix | `1****5` | range scan on whichever side is longer, filter the other; or BitmapAnd `d1` ∩ `d6` |
| Scattered / mixed | `1*3*5*` | `BitmapAnd` over the indexes for the fixed positions |
| `k ≤ 1` fixed | `*****7`, `******` | intentionally restricted for allocation (huge result set); allowed only for "count / browse" with a hard `LIMIT` |

### 4.3 Complexity

Let `k` = number of fixed digits, `N` = 10M, `m` = N / 10ᵏ = matches.

- **Prefix/suffix range scan:** O(log N + m) index entries — for `k ≥ 3`,
  `m ≤ 10k`, so **sub-millisecond to a few ms**.
- **BitmapAnd of `k` single-digit indexes:** each posting list ≈ N/10 = 1M
  entries; Postgres builds a bitmap and ANDs — **single-digit to low-tens of ms**
  at `k = 2`, faster as `k` grows.
- **`k ≤ 1`:** m ≈ 1M–10M → never fully materialised; `LIMIT`-bounded.

### 4.4 Optional in-memory accelerator — position/digit inverted index

For high search QPS, hold a **Roaring bitmap** `B[p][d]` for each of the
6 positions × 10 digits = **60 posting lists** of ticket IDs.

- **Build:** one scan of `tickets`; each ticket ID goes into exactly 6 bitmaps
  (one per position). Total ≈ 60M membership entries → **~50–100 MB** compressed.
- **Query:** `candidates = AND(B[p][pattern[p]] for each fixed position p)`.
  Roaring AND is `O(size/64)` word operations → **tens of microseconds** for
  `k ≥ 2`.
- **Freshness:** rebuild on startup; apply incremental changes via Postgres
  logical replication / CDC or `LISTEN/NOTIFY`. Staleness is safe: a candidate
  that no longer matches or is already taken simply fails the Postgres reserve
  and is skipped.
- **Sharding (if data grows 10×–100×):** partition bitmaps by `d1` across nodes;
  scatter-gather then union.

Data structures involved: **Roaring bitmaps** (compressed sorted sets) for the
inverted index; **B-tree / trie-like range scans** inside Postgres for
prefix/suffix; an **LRU cache** keyed by pattern for hot result sets (short TTL,
value = list of candidate IDs, *not* reservations).

---

## 5. Preventing duplicate simultaneous results

### 5.1 Invariant

> At any instant, a ticket has **at most one active holder**.
> "Active" = `status='sold'`, or `status='reserved'` with `reserved_until > now()`.

### 5.2 Primary mechanism — atomic search-and-reserve in one statement

```sql
WITH picked AS (
    SELECT id
    FROM tickets
    WHERE d5 = 2 AND d6 = 3                    -- pattern ****23
      AND ( status = 'available'
            OR (status = 'reserved' AND reserved_until <= now()) )
    ORDER BY id
    FOR UPDATE SKIP LOCKED                     -- ← the key line
    LIMIT :count
)
UPDATE tickets t
SET    status = 'reserved',
       reserved_by = :user_id,
       reserved_at = now(),
       reserved_until = now() + interval '5 minutes'
FROM   picked
WHERE  t.id = picked.id
RETURNING t.id, t.number;
```

Why this is correct:

- **`FOR UPDATE SKIP LOCKED`** — when user A and user B fire this concurrently
  for `****23`, A locks the first `:count` claimable rows; B *skips* those
  locked rows and locks the next ones. Neither blocks; neither can receive a
  row the other is reserving. The result sets are **disjoint by construction**.
- The **`UPDATE … RETURNING`** flips state in the same transaction, so a
  committed reservation is immediately visible and a rolled-back one leaves no
  trace.
- Concurrency across **multiple API instances** works unchanged — locks are held
  per database session, not per process.

### 5.3 Lease / reservation lifecycle

```
 available ──reserve──▶ reserved ──confirm──▶ sold
     ▲                     │
     └───── expire ────────┘   (reserved_until < now, or explicit release)
```

- **Reservation is a short lease** (`reserved_until`, e.g. 5 min). If the user
  doesn't complete the purchase, the row becomes claimable again — the
  `WHERE` predicate already treats an expired reservation as available, so
  correctness never depends on the cleanup job.
- **Reaper job** (every ~30 s) sets truly-expired rows back to
  `status='available'` and clears `reserved_*` — purely housekeeping / index
  hygiene.
- **Confirm:** `UPDATE tickets SET status='sold' WHERE id = :id AND reserved_by = :user AND reserved_until > now()` — the predicate stops a user confirming a lease that already expired and was re-allocated.

### 5.4 Defence in depth

| Guard | Protects against |
|---|---|
| Partial unique index: `CREATE UNIQUE INDEX active_reservation ON reservations(ticket_id) WHERE released_at IS NULL` | any code path that tries to create a second live reservation for a ticket |
| **Idempotency key** on the search-and-reserve request (unique in a `requests` table) | client ret/network retries producing double allocation |
| `reserved_by` + `reserved_until` re-checked on confirm | confirming a stolen/expired lease |
| Bound `:count` (e.g. ≤ 50) and require `k ≥ 2` | a single request draining a loose pattern |

### 5.5 Redis-based alternative for the lease layer

If reservation traffic outgrows what you want on the primary:

- Keep candidate generation in Postgres / the in-memory index.
- For each candidate ID, attempt `SET lease:{id} {user} NX EX 300` (Lua script
  to claim up to `:count` in one round trip). `NX` = atomic first-writer-wins;
  contention exists *only* on the exact tickets two users both try, and the
  loser just moves to the next candidate.
- Postgres still records the sale on confirm (source of truth).
- Trade-off: two stores to reason about, and a Redis failure needs a
  well-defined fallback (degrade to Postgres `SKIP LOCKED`).

---

## 6. Performance analysis & trade-offs

### 6.1 Expected latency (single modern node, 10M rows)

| Step | Postgres-only | With in-memory index |
|---|---|---|
| Candidate generation, `k ≥ 3` | 0.1–3 ms | ~10–50 µs |
| Candidate generation, `k = 2` | 3–30 ms | ~50–200 µs |
| Search-and-reserve `:count ≤ 50` | +0.5–3 ms | +0.5–3 ms (reserve still in PG) |
| Confirm | ~1 ms | ~1 ms |

### 6.2 Throughput & scaling

- **`SKIP LOCKED` scales near-linearly** with concurrent reservers for the same
  pattern, because they operate on disjoint rows — no lock convoy.
- Reads (pure search) scale out to **read replicas**; the primary is reserved
  for allocation writes.
- 10M → 1B rows: **hash-partition `tickets`** (by `id` or by `d1`); `SKIP LOCKED`
  works per partition. Shard the in-memory index by `d1`.
- API tier is stateless and scales horizontally with no new coordination.

### 6.3 Memory

- Postgres working set (data + indexes): ~1–2 GB, RAM-resident.
- Optional Roaring index: ~50–100 MB per node.
- Hot-pattern LRU cache: bounded (e.g. 10k patterns × few KB).

### 6.4 Trade-offs made

| Choice | Upside | Cost |
|---|---|---|
| On-demand search + hot cache (vs. precompute all patterns) | no write amplification; fresh results | first hit on a cold pattern pays the scan |
| Denormalised `d1..d6` + `number_rev` | every wildcard class gets an index | ~10 B/row extra, generated on write |
| In-memory index is advisory only | can lag PG without risking correctness | a stale candidate wastes one reserve attempt |
| Lease TTL (vs. hold until user acts) | abandoned carts self-heal; pool stays liquid | a slow user can lose their reservation; pick TTL accordingly |
| Postgres for allocation (vs. Redis-first) | one durable source of truth, simplest correct design | primary carries reservation write load (mitigate via partitioning) |
| Require `k ≥ 2` for allocation | keeps result sets and lock counts sane | users can't allocate against ultra-loose patterns (reasonable product rule) |

### 6.5 Failure modes

| Failure | Behaviour |
|---|---|
| API pod crashes mid-reservation | transaction rolls back; rows never left locked; nothing to clean |
| User abandons after reserving | lease expires → ticket returns to pool automatically |
| Reaper job down | no correctness impact (predicate handles expiry); only index bloat until it resumes |
| In-memory index node down | fall back to Postgres candidate generation |
| Postgres primary failover | standard HA (Patroni); in-flight uncommitted reservations are lost (correct) |
| Clock skew between app and DB | use `now()` **in the SQL** (DB clock) for all lease math — never the app clock |

---

## 7. Summary

| Requirement | Solution |
|---|---|
| Search 10M tickets by 6-char digit/`*` pattern | Postgres with per-digit + prefix + reversed-suffix indexes; optional in-memory Roaring-bitmap inverted index for µs-level candidate generation |
| Performance at scale | data fits in RAM; `k ≥ 3` queries are sub-ms; partition + read replicas for growth |
| No duplicate ticket to two users at once | **`SELECT … FOR UPDATE SKIP LOCKED`** + `UPDATE … RETURNING` in one statement → concurrent requests get disjoint rows; backed by a partial unique index and idempotency keys |
| Allocation without a global pattern lock | per-row locking only; loose patterns never serialise users against each other |
| Abandoned reservations | short TTL lease; `WHERE` predicate self-heals; reaper is just housekeeping |
| Production practicality | one primary database, well-understood ops, stateless API tier, optional Redis/in-memory accelerators that cannot compromise correctness |
