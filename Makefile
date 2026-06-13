.PHONY: all ingest load clean

# Default target runs the entire pipeline
all: load

# Alias for convenience - run 'make ingest' to trigger extraction and transformation
ingest: data/dataset.csv

# Make will only run this if data/dataset.csv is missing or cmd/ingest/main.go has changed.
data/dataset.csv: cmd/ingest/main.go
	@echo "==> Extracting and Transforming data..."
	@mkdir -p data
	go run cmd/ingest/main.go

# Loading depends on the CSV artifact.
load: data/dataset.csv
	@echo "==> Loading data into PostgreSQL..."
	go run cmd/load/main.go

# Remove generated data artifacts
clean:
	@echo "==> Cleaning up artifacts..."
	rm -f data/dataset.csv
