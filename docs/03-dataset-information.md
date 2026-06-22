# Dataset Information

## Source

This project uses the [AOL Search Logs Dataset](https://archive.org/download/academictorrents_cd339bddeae7126bb3b15f3a72c903cb0c401bd1/AOL_search_data_leak_2006.zip), a collection of real searches made by people in the United States between March 1, 2006 and May 31, 2006.

**Key Statistics:**

| Metric | Value |
| :--- | :--- |
| **Search Queries** | ~20 million |
| **Unique Users** | ~650,000 (anonymized) |
| **Time Period** | March – May 2006 (3 months) |
| **Source** | AOL Research (via archive.org) |

**Download URL:**
```
https://archive.org/download/academictorrents_cd339bddeae7126bb3b15f3a72c903cb0c401bd1/AOL_search_data_leak_2006.zip
```

## Raw Format

The raw data is a **ZIP archive** containing multiple gzip-compressed text files. Each text file contains tab-separated rows representing individual search events.

**Archive Structure:**
```
AOL_search_data_leak_2006.zip
└── AOL-user-ct-collection/
    ├── user-ct-test-collection-01.txt.gz
    ├── user-ct-test-collection-02.txt.gz
    ├── ...
    └── user-ct-test-collection-10.txt.gz
```

**File Pattern (regex):** `^AOL-user-ct-collection/user-ct-test-collection-\d\d\.txt\.gz$`

**TSV Columns:**

| Column Index | Column Name | Description |
| :--- | :--- | :--- |
| 0 | AnonID | An anonymized user identifier |
| 1 | **Query** | The search query string **(only column used)** |
| 2 | QueryTime | The date and time of the search |
| 3 | ItemRank | The rank of the item clicked (if any) |
| 4 | ClickURL | The URL of the item clicked (if any) |

> **Note:** Only column index 1 (`Query`) is extracted by the ingestion pipeline. All other columns are discarded.

## Transformation Pipeline

The `cmd/ingest/main.go` program performs the following steps:

| Step | Action | Implementation |
| :--- | :--- | :--- |
| 1. Download | HTTP GET to archive.org | `http.Get(DatasetSourceURL)` |
| 2. Load into Memory | Read entire ZIP | `io.ReadAll(response.Body)` — required because ZIP needs random access |
| 3. Open ZIP | Create reader | `zip.NewReader(readerAt, size)` |
| 4. Filter Files | Match pattern | `datasetFilePattern.MatchString(file.Name)` |
| 5. Decompress | Gzip stream | `gzip.NewReader(zipDecompressed)` |
| 6. Parse | Line-by-line TSV | `bufio.NewScanner` + `strings.Split(line, "\t")` |
| 7. Extract | Column index 1 | `fields[1]` — the raw query string (no normalization) |
| 8. Aggregate | Count frequencies | `map[string]int` — increments count per unique query |
| 9. Write CSV | Output file | `csv.Writer` to `data/dataset.csv` |

> **Important:** Unlike some ETL pipelines, this one does **not** normalize queries (no lowercasing, no trimming, no typo correction). Queries are stored exactly as typed by users. Typo normalization is not part of the MVP but can be included in future requirements.

## Output Format

The resulting `data/dataset.csv` file is a standard CSV with a header row:

```csv
Query,Frequency
google,2841
myspace.com,2520
www.google.com,2016
...
```

| Property | Value |
| :--- | :--- |
| **File** | `data/dataset.csv` |
| **Format** | CSV (comma-separated) |
| **Header** | `Query,Frequency` |
| **Sort Order** | Non-deterministic (Go map iteration order) |
| **File Size** | ~256 MB |
| **Git Status** | Ignored (listed in `.gitignore`) |

## Regenerating the Dataset

```bash
# Generate (or regenerate) the dataset
make ingest

# Clean up
make clean    # Deletes data/dataset.csv
```

## Historical Note

The dataset was released by AOL Research for academic purposes in 2006 but caused a massive privacy scandal. Even though user IDs were anonymized, journalists and researchers were able to cross-reference search queries to identify specific individuals. This incident became a landmark case in data privacy discussions.
