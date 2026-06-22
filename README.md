# Typeahead System

[![Typeahead Demo](./docs/demo.png)](./docs/demo.png)

A high-performance typeahead (search autocomplete) system designed to predict and suggest the most popular search queries as a user types. Built as an assignment for a High-Level Design (HLD) course, focusing on system scalability, low-latency requirements, and efficient data ingestion.

## Problem Statement

Design a typeahead application that:
- Provides real-time suggestions as users type.
- Shows the top 10 most relevant suggestions based on prefix matching and historical query frequency.
- Optimizes for **low latency** and **high availability**, even at the cost of eventual consistency.
- Handles a massive scale: ~2 million requests per second and ~320 TB of data over 20 years.

## Documentation

For a detailed breakdown of the requirements, design decisions, and data modeling, refer to the `docs/` directory:

| Document | Description |
| :--- | :--- |
| [01 Requirements and Scale](./docs/01-requirements-and-scale.md) | Problem statement, functional/non-functional requirements, and scale estimations |
| [02 High Level Design](./docs/02-high-level-design.md) | Trie vs. Key-Value store, sharding strategies, batching, and serving architecture |
| [03 Dataset Information](./docs/03-dataset-information.md) | Overview of the AOL Search Logs dataset |
| [04 ETL Pipeline Design](./docs/04-etl-pipeline-design.md) | Architecture of the ingestion and loading pipeline |

## Architecture Overview

```
┌──────────────────────────────────────────────────────────────────────────────┐
│                        Data Ingestion (Batch ETL)                           │
│                                                                              │
│  AOL Logs (.zip)  ──▶  cmd/ingest  ──▶  data/dataset.csv                    │
│  (archive.org)                              │                                │
│                                  ┌──────────┴──────────┐                     │
│                                  ▼                     ▼                     │
│                           cmd/load-redis        cmd/load-postgres            │
│                                  │                     │                     │
│                                  ▼                     ▼                     │
│                             Redis (ZSETs)        PostgreSQL                  │
└──────────────────────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────────────────────┐
│                        Serving Layer                                         │
│                                                                              │
│  Browser (web/)  ──▶  cmd/server (:8080)  ──▶  internal/store                │
│                       ┌───────────────────┐       │                          │
│                       │ GET  /api/v1/     │    ┌──┴──┐                       │
│                       │   health          │    │     │                       │
│                       │   suggestions     │  Redis  Postgres                 │
│                       │ POST /api/v1/     │  (ZSET)  (SQL)                   │
│                       │   search          │                                  │
│                       └───────────────────┘                                  │
└──────────────────────────────────────────────────────────────────────────────┘
```

## Tech Stack

| Layer | Technology |
| :--- | :--- |
| **Language** | Go 1.26+ |
| **Serving Store** | Redis (Sorted Sets via `go-redis/v9`) |
| **Analytics Store** | PostgreSQL (via `pgx/v5` and `pgxpool`) |
| **Frontend** | HTML, CSS, JavaScript |
| **Infrastructure** | Docker & Docker Compose |
| **Orchestration** | Makefile |

## Project Structure

```
.
├── cmd/
│   ├── bench/              # Load-testing / benchmarking (placeholder)
│   ├── ingest/             # ETL: download, parse, aggregate → CSV
│   │   └── main.go
│   ├── load-postgres/      # Bulk-load CSV into PostgreSQL (COPY FROM)
│   │   └── main.go
│   ├── load-redis/         # Load CSV into Redis Sorted Sets (ZADD)
│   │   └── main.go
│   └── server/             # HTTP API server (dual-backend)
│       └── main.go
├── internal/
│   └── store/              # TypeaheadStore interface + implementations
│       ├── store.go        # Interface: GetSuggestions, IncrementFrequency
│       ├── postgres/       # PostgreSQL implementation (pgxpool)
│       │   └── postgres.go
│       └── redis/          # Redis implementation (ZSET)
│           └── redis.go
├── web/                    # Static frontend assets
│   ├── index.html
│   ├── styles.css
│   └── app.js
├── data/                   # Generated dataset (git-ignored)
│   └── dataset.csv
├── docs/                   # Design documents and diagrams
├── docker-compose.yml      # Redis service
├── Makefile                # Build & pipeline orchestration
├── go.mod
└── go.sum
```

## Local Setup

### Prerequisites

- Go 1.26+
- Docker & Docker Compose
- Make

### 1. Start Redis

Spin up a Redis instance using Docker Compose:

```bash
docker compose up -d
```

This starts Redis on `localhost:6379` with persistent storage via a named volume.

### 2. Run the ETL Pipeline

The project uses a `Makefile` to manage the ingestion and loading of the AOL dataset.

```bash
# Step 1: Download, parse, and aggregate the dataset into CSV
make ingest

# Step 2: Load the CSV into Redis Sorted Sets
make load-redis
```

> **Note:** The ingest step downloads a ~200MB ZIP file from archive.org and loads it entirely into memory (ZIP requires random access). Ensure sufficient RAM.

### 3. Start the Server

The server requires a `STORE_TYPE` environment variable to select the storage backend:

```bash
# Using Redis backend (recommended)
STORE_TYPE=redis go run cmd/server/main.go

# Using PostgreSQL backend
STORE_TYPE=postgres go run cmd/server/main.go
```

The server starts on [http://localhost:8080](http://localhost:8080).

### 4. Open the Frontend

The web frontend is a standalone HTML page that communicates with the API server.

Open `web/index.html` directly in your browser, or navigate to [http://localhost:8080](http://localhost:8080).

> **Note:** The frontend currently hardcodes `http://localhost:8080` as the API base URL.

- Start typing to see autocomplete suggestions.
- Suggestions with length ≥50 characters are filtered out on the frontend.
- Press **Enter** to submit a search, which increments the query's frequency.

## API Reference

### `GET /api/v1/health`

Health check endpoint.

**Response:** `200 OK` with body `OK`

### `GET /api/v1/suggestions`

Returns autocomplete suggestions for a given prefix.

**Query Parameters:**

| Parameter | Type | Required | Description |
| :--- | :--- | :--- | :--- |
| `prefix` | string | Yes | Search prefix |
| `limit` | integer | Yes | Maximum number of suggestions to return |

**Response:**

```json
["google maps", "google earth", "google"]
```

> Returns a JSON array of suggestion strings, ordered by descending frequency.

**Error Responses:**

| Status | Condition |
| :--- | :--- |
| Empty response | Missing `prefix` or `limit <= 0` |
| `500 Internal Server Error` | Store query failure |

**Behavior Notes:**
- Gracefully handles aborted requests (`context.Canceled`)
- CORS enabled for all origins

### `POST /api/v1/search`

Increments the frequency of a search query.

**Request Body:** Plain text query string

**Response:** `201 Created`

**Backend Behavior:**
- **Redis:** Increments the query's score by 1 in sorted sets for every prefix of the query (from 1 char to the full query length)
- **PostgreSQL:** Inserts a new row with frequency 1, or increments existing frequency by 1

## Alternative: PostgreSQL Backend

The project also includes a PostgreSQL storage backend:

```bash
# Start a PostgreSQL container
docker run -d \
  --name frequency-db \
  -e POSTGRES_USER=postgres \
  -e POSTGRES_PASSWORD=admin123 \
  -e POSTGRES_DB=frequency-db \
  -p 5432:5432 \
  postgres:16

# Load data into PostgreSQL
make load-postgres

# Start server with PostgreSQL backend
STORE_TYPE=postgres go run cmd/server/main.go

# Verify loaded data
docker exec -it frequency-db psql -U postgres -d frequency-db -c "SELECT * FROM search_queries LIMIT 10;"
```

**PostgreSQL `GetSuggestions` implementation:** Uses `SELECT query FROM search_queries WHERE query LIKE '<prefix>%' ORDER BY frequency DESC LIMIT <limit>`.

**PostgreSQL `IncrementFrequency` implementation:** Performs a SELECT to check current frequency, then either INSERTs a new row or UPDATEs the existing one.

> **Note:** The PostgreSQL connection string is hardcoded in both `cmd/load-postgres/main.go` and `cmd/server/main.go` as `postgresql://postgres:admin123@localhost:5432/frequency-db`.

## Cleanup

To remove generated artifacts and stop services:

```bash
# Remove the generated dataset
make clean

# Stop Redis
docker compose down

# (If using PostgreSQL)
docker stop frequency-db && docker rm frequency-db
```
