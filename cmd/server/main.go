package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5"
)

const (
	SERVER_ADDR string = "localhost:8080"
)

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/v1/health", healthCheckHandler)
	mux.HandleFunc("GET /api/v1/suggestions", suggestionsHandler)

	fmt.Printf("Server is running on http://%s\n", SERVER_ADDR)
	if err := http.ListenAndServe(SERVER_ADDR, mux); err != nil {
		fmt.Fprintf(os.Stderr, "error starting HTTP server: %v", err)
		os.Exit(1)
	}
}

func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func suggestionsHandler(w http.ResponseWriter, r *http.Request) {
	prefix := r.URL.Query().Get("prefix")

	ctx := r.Context()

	conn, err := pgx.Connect(ctx, os.Getenv("FREQUENCY_DB_URL"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "/suggestions: failed to connect to frequency db: %v\n", err)
		http.Error(w, "Unable to connect to database", http.StatusInternalServerError)
		return
	}

	query := fmt.Sprintf(`SELECT query FROM search_queries WHERE query LIKE %s ORDER BY frequency DESC LIMIT 10;`, "'"+prefix[1:len(prefix)-1]+"%"+"'")
	fmt.Println(query)

	rows, err := conn.Query(ctx, query)
	if err != nil {
		fmt.Fprintf(os.Stderr, "/suggestions: error executing sql read query for fetching suggestions: %v\n", err)
		http.Error(w, "Error fetching suggestions", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	queries := make([]string, 0, 10)

	for rows.Next() {
		var query string
		rows.Scan(&query)
		fmt.Println(query)
		queries = append(queries, query)
	}
	if err := rows.Err(); err != nil {
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
