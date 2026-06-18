package main

import (
	"context"
	"encoding/json"
	"fmt"
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
