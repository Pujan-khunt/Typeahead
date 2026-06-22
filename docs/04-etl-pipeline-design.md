# ETL Pipeline Design - AOL Search Logs

This document outlines the design for the ETL (Extract, Transform, Load) pipeline for the AOL search logs dataset. The pipeline is designed as a one-off sequential process, with two supported storage backends: **Redis** (primary, for serving) and **PostgreSQL** (secondary, for analytics).

## 1. Project Structure

The project follows the standard Go layout to separate command-line entry points from data and orchestration.

```text
.
├── Makefile                   # Orchestrator for the pipeline
├── cmd/
│   ├── ingest/                # Step 1: Extract & Transform
│   │   └── main.go
│   ├── load-redis/            # Step 2a: Load into Redis
│   │   └── main.go
│   └── load-postgres/         # Step 2b: Load into PostgreSQL
│       └── main.go
└── data/
    └── dataset.csv            # Intermediate artifact (Generated)
```

## 2. Pipeline Stages

```
┌──────────────────────────────────────┐
│     Raw AOL Search Logs              │
│     (ZIP on archive.org)             │
└─────────────────┬────────────────────┘
                  │
┌─────────────────▼────────────────────┐
│      1. EXTRACT                      │
│  HTTP Download ZIP → Read All        │
│  into Memory (ZIP needs random       │
│  access) → Open each .txt.gz        │
│  (cmd/ingest/main.go)                │
└─────────────────┬────────────────────┘
                  │
┌─────────────────▼────────────────────┐
│      2. TRANSFORM                    │
│  Gzip Decompress → Parse TSV →       │
│  Extract Query Column (index 1)      │
│  → Aggregate in map[string]int       │
│  (cmd/ingest/main.go)                │
└─────────────────┬────────────────────┘
                  │
┌─────────────────▼────────────────────┐
│      3. LOAD TO CSV                  │
│  Write to data/dataset.csv           │
│  Columns: Query, Frequency           │
│  (cmd/ingest/main.go)                │
└─────────────────┬────────────────────┘
                  │
         ┌────────┴────────┐
         ▼                 ▼
┌────────────────┐  ┌──────────────────┐
│ 4a. LOAD       │  │ 4b. LOAD         │
│ REDIS          │  │ POSTGRESQL       │
│                │  │                  │
│ • For each     │  │ • CREATE TABLE   │
│   query, ZADD  │  │   IF NOT EXISTS  │
│   to sorted    │  │   search_queries │
│   set for      │  │ • COPY FROM STDIN│
│   every prefix │  │   (bulk load)    │
│ • Key = each   │  │                  │
│   prefix of    │  │ (cmd/load-       │
│   the query    │  │  postgres/       │
│ • Score =      │  │  main.go)        │
│   frequency    │  │                  │
│                │  │                  │
│ (cmd/load-     │  │                  │
│  redis/        │  │                  │
│  main.go)      │  │                  │
└────────────────┘  └──────────────────┘
```

## 3. Component Details

### Stage 1-3: Ingest (`cmd/ingest/main.go`)

A single Go program that handles extraction, transformation, and CSV generation.

| Step | Action | Details |
| :--- | :--- | :--- |
| Extract | HTTP GET + read all into memory | Downloads the AOL ZIP from `archive.org`. ZIP requires random access, so the entire file is loaded into memory via `io.ReadAll`. |
| Extract | `zip.NewReader` | Opens the ZIP archive and iterates over files matching `user-ct-test-collection-XX.txt.gz` |
| Extract | `gzip.NewReader` | Each matched file is gzip-compressed; decompressed via streaming |
| Transform | `bufio.Scanner` + `strings.Split` | Reads line-by-line, splits on tab, extracts column index 1 (the query) |
| Transform | `map[string]int` | Aggregates frequency of each unique query in memory |
| Load | `csv.Writer` | Writes `Query,Frequency` CSV to `data/dataset.csv` |

**Source URL:** `https://archive.org/download/academictorrents_cd339bddeae7126bb3b15f3a72c903cb0c401bd1/AOL_search_data_leak_2006.zip`

**File Pattern:** `^AOL-user-ct-collection/user-ct-test-collection-\d\d\.txt\.gz$`

**Output:** `data/dataset.csv` containing unique queries with their aggregated frequencies.

### Stage 4a: Load Redis (`cmd/load-redis/main.go`)

Loads the CSV into Redis Sorted Sets, which is the **primary serving store** used by the API server.

| Aspect | Details |
| :--- | :--- |
| **Key Schema** | Every prefix of the query (from 1 char to the full query). E.g., for "google": keys are `g`, `go`, `goo`, `goog`, `googl`, `google` |
| **Data Structure** | Redis Sorted Set (`ZSET`) |
| **Member** | The full query string |
| **Score** | The query's search frequency (as `float64`) |
| **Progress** | Prints progress every 1,000 records |
| **Connection** | `localhost:6379`, no password, DB 0, Protocol 2 |

> **Note:** The header row (`Query,Frequency`) from the CSV is also processed but would result in an entry with a non-numeric frequency (Atoi returns 0).

### Stage 4b: Load PostgreSQL (`cmd/load-postgres/main.go`)

Loads the CSV into a PostgreSQL table for analytics or as an alternative storage backend.

| Aspect | Details |
| :--- | :--- |
| **Table** | `search_queries` (created with `CREATE TABLE IF NOT EXISTS`) |
| **Schema** | `query TEXT PRIMARY KEY, frequency BIGINT NOT NULL DEFAULT 0` |
| **Method** | `pgx.PgConn().CopyFrom` — PostgreSQL's `COPY FROM STDIN` protocol with `FORMAT csv, HEADER true` |
| **Connection** | Hardcoded: `postgresql://postgres:admin123@localhost:5432/frequency-db` |

## 4. Orchestration (Makefile)

A `Makefile` manages the dependencies between the stages.

```makefile
.PHONY: all ingest load-postgres load-redis clean

all: load

data/dataset.csv: cmd/ingest/main.go
	go run cmd/ingest/main.go

load-postgres: data/dataset.csv
	go run cmd/load-postgres/main.go

load-redis: data/dataset.csv
	go run cmd/load-redis/main.go

clean:
	rm -f data/dataset.csv
```

### Targets:

| Target | Description | Dependencies |
| :--- | :--- | :--- |
| `all` | Default target (currently references `load` which is undefined — should be `load-redis`) | `load` |
| `data/dataset.csv` | Runs the ingest step to produce the CSV | `cmd/ingest/main.go` |
| `load-redis` | Loads CSV into Redis Sorted Sets | `data/dataset.csv` |
| `load-postgres` | Loads CSV into PostgreSQL via COPY | `data/dataset.csv` |
| `clean` | Removes `data/dataset.csv` | — |

> **Note:** The `all` target references `load` which does not exist as a Makefile target. Running `make` without arguments will fail. Use `make load-redis` or `make load-postgres` explicitly.

## 5. Design Decisions

1. **In-Memory ZIP Processing:** The entire ZIP file is loaded into memory because Go's `archive/zip` package requires `io.ReaderAt` (random access). This is feasible because the ZIP file is ~200MB.

2. **In-Memory Aggregation:** The entire frequency map is held in memory during the Transform stage. This is feasible because the unique query set fits comfortably in RAM.

3. **Unsorted Output:** Unlike a typical production pipeline, the CSV output is **not sorted** — it iterates over a Go map which has non-deterministic order.

4. **Dual Storage Backends:** Redis Sorted Sets are used for the low-latency serving path, while PostgreSQL provides a relational store for analytics or alternative serving.

5. **Per-Prefix Redis Keys:** The Redis loader creates a sorted set for **every prefix** of every query (not just a fixed-length prefix). This enables O(1) lookups for any prefix length but uses significantly more memory.

6. **Bulk PostgreSQL Copy:** `CopyFrom` uses PostgreSQL's `COPY` protocol, which is orders of magnitude faster than individual `INSERT` statements for large datasets.

7. **No Query Normalization:** Queries are stored exactly as they appear in the raw data — no lowercasing, trimming, or typo correction is applied during ingestion.
