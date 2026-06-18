package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	SERVER_ADDR string = "localhost:8080"
)

type App struct {
	DB *pgxpool.Pool
}

func main() {
	pool, err := pgxpool.New(context.Background(), os.Getenv("FREQUENCY_DB_URL"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error connecting to database: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()

	app := App{DB: pool}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/health", app.healthCheckHandler)
	mux.HandleFunc("GET /api/v1/suggestions", app.suggestionsHandler)
	mux.HandleFunc("POST /api/v1/search", app.searchHandler)

	globalHandler := corsMiddleware(mux)

	fmt.Printf("Server is running on http://%s\n", SERVER_ADDR)
	if err := http.ListenAndServe(SERVER_ADDR, globalHandler); err != nil {
		fmt.Fprintf(os.Stderr, "error starting HTTP server: %v\n", err)
		os.Exit(1)
	}
}

func (app *App) healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		headers := w.Header()
		headers.Set("Access-Control-Allow-Origin", "*")
		headers.Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		headers.Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		next.ServeHTTP(w, r)
	})
}

func (app *App) suggestionsHandler(w http.ResponseWriter, r *http.Request) {
	prefix := r.URL.Query().Get("prefix")

	if len(prefix) == 0 {
		return
	}

	ctx := r.Context()
	cleanPrefix := prefix[1:len(prefix)-1] + "%"
	query := "SELECT query FROM search_queries WHERE query LIKE $1 ORDER BY frequency DESC LIMIT 10;"

	rows, err := app.DB.Query(ctx, query, cleanPrefix)
	if err != nil && err != context.Canceled {
		fmt.Fprintf(os.Stderr, "/suggestions: error executing sql read query for fetching suggestions: %v\n", err)
		http.Error(w, "Error fetching suggestions", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	queries := make([]string, 0, 10)

	for rows.Next() {
		var query string
		rows.Scan(&query)
		queries = append(queries, query)
	}
	if err := rows.Err(); err != nil && err != context.Canceled {
		fmt.Fprintf(os.Stderr, "/suggestions: error reading suggestions from database: %v\n", err)
		http.Error(w, "Error reading values", http.StatusInternalServerError)
		return
	}

	output, err := json.Marshal(queries)
	if err != nil {
		fmt.Fprintf(os.Stderr, "/suggestions: error converting queries into json: %v\n", err)
		http.Error(w, "Error transforming output", http.StatusInternalServerError)
		return
	}

	w.Write(output)
}

func (app *App) searchHandler(w http.ResponseWriter, r *http.Request) {
	// Read the search query from user.
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "/search: error reading search query from body: %v\n", err)
		http.Error(w, "Error reading query", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	ctx := r.Context()
	query := string(raw)

	// Retrieve the existing frequency of the search query.
	row := app.DB.QueryRow(ctx, `SELECT frequency FROM search_queries WHERE query = $1`, query)
	var frequency int64
	row.Scan(&frequency)

	// Determine type of query: insert/update based on frequency.
	var updateQuery string
	if frequency == 0 {
		updateQuery = "INSERT INTO search_queries (query, frequency) VALUES ($1, $2);"
	} else {
		updateQuery = "UPDATE search_queries SET frequency = $2 WHERE query = $1;"
	}

	// Execute update query.
	app.DB.Exec(ctx, updateQuery, query, frequency+1)

	// Notify client about resource creation.
	w.WriteHeader(http.StatusCreated)
}
