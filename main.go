package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/miguelbernadi/dashboard/daterange"
	"github.com/miguelbernadi/dashboard/provider"
	"github.com/miguelbernadi/dashboard/provider/fakeprovider"
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

var providers = []provider.Provider{
	fakeprovider.FakeProvider{},
}

var queries provider.QueryList

func genQueries(list provider.QueryList) <-chan QueryInput {
	out := make(chan QueryInput)
	go func() {
		for k, f := range list {
			out <- QueryInput{k, f}
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
	t, ok := daterange.FromContext(ctx)
	if !ok {
		log.Println("No dates present in context. Aborting.")
		close(out)
		return out
	}
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
	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}

func query(w http.ResponseWriter, r *http.Request) {
	defer startTimer("search")()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Parameter parsing
	ctx, err := daterange.NewContextFromRequest(ctx, r)
	if err != nil {
		log.Println(err)
		cancel()
	}

	result := make(provider.ResultList)
	// Gather results
	for e := range runQueries(ctx, genQueries(queries)) {
		result = result.Append(e)
	}

	// Display results
	_, err = io.WriteString(w, fmt.Sprintf("%# v", result))
	if err != nil {
		log.Println(err)
		cancel()
	}
}

func welcome(w http.ResponseWriter, r *http.Request) {
	_, err := io.WriteString(w, "Hello World!")
	if err != nil {
		log.Panic(err)
	}
}

func main() {
	server := http.Server{
		Addr: ":8080",
	}

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

	log.Println("Starting server on port 8080")
	http.HandleFunc("/", welcome)
	http.HandleFunc("/search", query)

	err := server.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}
