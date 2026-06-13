# ETL Pipeline Design - AOL Search Logs

This document outlines the design for the ETL (Extract, Transform, Load) pipeline for the AOL search logs dataset. The pipeline is designed as a one-off sequential process for a high-level design course assignment.

## 1. Project Structure

The project follows the standard Go layout to separate command-line entry points from data and orchestration.

```text
.
├── Makefile               # Orchestrator for the pipeline
├── cmd/
│   ├── ingest/            # Step 1: Extract & Transform
│   │   └── main.go
│   └── load/              # Step 2: Load
│       └── main.go
├── data/
│   └── dataset.csv        # Intermediate artifact (Generated)
└── .env                   # Environment configuration (Postgres URL)
```

## 2. Pipeline Stages

### Stage 1: Ingest (Extract & Transform)
- **Component:** `cmd/ingest/main.go`
- **Responsibility:**
    - Download the AOL search logs ZIP from the source URL.
    - Extract and decompress the `.txt.gz` files.
    - Parse queries and calculate frequencies.
    - Export the unique queries and their frequencies to `data/dataset.csv`.
- **Artifact:** `data/dataset.csv`

### Stage 2: Load (Load)
- **Component:** `cmd/load/main.go`
- **Responsibility:**
    - Establish a connection to the PostgreSQL database.
    - Create the `search_queries` table if it does not exist.
    - Perform a high-performance bulk copy of `data/dataset.csv` into the database using the PostgreSQL `COPY` protocol via `pgx`.
- **Constraint:** Depends on the existence of `data/dataset.csv`.

## 3. Orchestration (Makefile)

A `Makefile` will manage the dependencies between the stages. It ensures that data is transformed before it is loaded and avoids redundant execution if the code has not changed.

### Targets:
- `all`: The default target, runs the full pipeline.
- `data/dataset.csv`: Runs the ingestion script to produce the CSV file. Depends on `cmd/ingest/main.go`.
- `load`: Runs the loading script. Depends on `data/dataset.csv`.
- `clean`: Removes generated data artifacts.

## 4. Error Handling & Validation
- Standard Go error checking for network requests, file I/O, and database operations.
- The `load` step will skip the header line of the CSV to prevent type mismatches.
- PostgreSQL constraints (PRIMARY KEY) will ensure data integrity during the bulk load.
