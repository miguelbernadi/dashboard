package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/miguelbernadi/dashboard/daterange"
	"github.com/miguelbernadi/dashboard/provider"
	"github.com/miguelbernadi/dashboard/provider/postgresprovider"
)

func startTimer(name string) func() {
	t := time.Now()
	log.Println(name, "started")
	return func() {
		d := time.Now().Sub(t)
		log.Println(name, "took", d)
	}
}

// QueryInput is a structure for query processing
type QueryInput struct {
	k string
	f provider.QueryFunction
}

var providers []provider.Provider

var queries provider.QueryList

func genQueries(ctx context.Context, list provider.QueryList) <-chan QueryInput {
	out := make(chan QueryInput)
	go func() {
		for k, f := range list {
			select {
			case <-ctx.Done():
				log.Println("genQueries was cancelled")
				close(out)
				return
			case out <- QueryInput{k, f}:
			}
		}
		close(out)
	}()
	return out
}

func runQueries(
	ctx context.Context,
	in <-chan QueryInput,
) <-chan provider.ResultList {
	out := make(chan provider.ResultList)
	var wg sync.WaitGroup
	wg.Add(1)

	t, ok := daterange.FromContext(ctx)
	if !ok {
		log.Println("No dates present in context. Aborting.")
		close(out)
		return out
	}
	go func() {
		for q := range in {
			wg.Add(1)
			go func(q QueryInput) {
				stop := startTimer(q.k)
				res, err := q.f(ctx, t.Begin, t.End)
				if err != nil {
					log.Printf(
						"Query %s failed with error %s\n",
						q.k,
						err.Error(),
					)
				}
				out <- res
				wg.Done()
				stop()
			}(q)
		}
	}()
	go func() {
		wg.Wait()
		close(out)
	}()
	wg.Done()
	return out
}

func query(w http.ResponseWriter, r *http.Request) {
	defer startTimer("search")()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if r.Method != "POST" {
		http.Error(w, "Operation not supported", http.StatusBadRequest)
		return
	}

	// Parameter parsing
	ctx, err := daterange.NewContextFromRequest(ctx, r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		log.Println(err)
		return
	}

	// Gather results
	result := make(provider.ResultList)
	for e := range runQueries(ctx, genQueries(ctx, queries)) {
		result = result.Append(e)
	}

	// Display results
	if r.Header.Get("Accept") == "application/json" {
		w.Header().Set("Content-Type", "application/json")
		message, err := json.Marshal(result)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			log.Println(err)
			return
		}
		_, err = io.WriteString(w, string(message))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			log.Println(err)
			return
		}
	} else {
		_, err = io.WriteString(w, fmt.Sprintf("%# v", result))
		if err != nil {
			log.Println(err)
			return
		}
	}
}

func main() {
	server := http.Server{
		Addr: ":8080",
	}

	postgres, err := postgresprovider.Setup(
	)
	if err != nil {
		log.Fatal(err)
	}
	providers = append(providers, postgres)

	stop := startTimer("Login to Data Providers")
	queries = make(provider.QueryList)
	// Log into data providers
	for _, provider := range providers {
		err := provider.Login()
		if err != nil {
			log.Fatal("Error logging in.", err.Error())
		}
		q, err := provider.Register()
		if err != nil {
			log.Fatal(err)
		}
		for k, f := range q {
			queries[k] = f
		}
	}
	stop()

	// Serve static files
	fs := http.FileServer(http.Dir("dist"))

	log.Println("Starting server on port 8080")
	http.Handle("/", fs)
	http.HandleFunc("/search", query)

	err = server.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}
