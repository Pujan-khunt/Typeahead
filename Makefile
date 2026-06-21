.PHONY: all ingest load-postgres clean

all: load

data/dataset.csv: cmd/ingest/main.go
	@echo "==> Extracting and Transforming data..."
	@mkdir -p data
	go run cmd/ingest/main.go

load-postgres: data/dataset.csv
	@echo "==> Loading data into PostgreSQL..."
	go run cmd/load-postgres/main.go

clean:
	@echo "==> Cleaning up artifacts..."
	rm -f data/dataset.csv
