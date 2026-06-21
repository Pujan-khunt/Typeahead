package main

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"

	"github.com/redis/go-redis/v9"
)

func main() {
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		Protocol: 2,
		DB:       0,
	})

	f, err := os.Open("data/dataset.csv")
	if err != nil {
		log.Fatalf("Unable to open dataset: %v", err)
	}

	r := csv.NewReader(f)
	for {
		record, err := r.Read()

		if errors.Is(err, io.EOF) {
			fmt.Println("All records have been processed.")
			return
		}
		if err != nil {
			fmt.Printf("Error reading record: %v\n", err)
			continue
		}

		query := record[0]
		freq, _ := strconv.Atoi(record[1])

		err = client.Set(context.Background(), query, freq, 0).Err()
		if err != nil {
			panic(err)
		}
	}
}
