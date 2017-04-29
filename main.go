package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"
)

func startTimer(name string) func() {
	t := time.Now()
	log.Println(name, "started")
	return func() {
		d := time.Now().Sub(t)
		log.Println(name, "took", d)
	}
}

// ResultList is a list of query results
type ResultList map[string]interface{}

func (r ResultList) Append(s ResultList) ResultList {
	for k, v := range s {
		r[k] = v
	}
	return r
}

// QueryFunction is a function performing a query
type QueryFunction func(
	ctx context.Context, date1, date2 time.Time,
) (ResultList, error)

// QueryList represents a list of QueryFunctions
type QueryList map[string]QueryFunction

// QueryInput is a structure for query processing
type QueryInput struct {
	k string
	f QueryFunction
}

// Provider must be satisfied by data providers
type Provider interface {
	Login() error
	Register() (QueryList, error)
}

var providers = []Provider{
	FakeProvider{},
}

var queries QueryList

func genQueries(list QueryList) <-chan QueryInput {
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
	begin, end time.Time,
	in <-chan QueryInput,
) <-chan ResultList {
	out := make(chan ResultList)
	var wg sync.WaitGroup
	for q := range in {
		wg.Add(1)
		go func(q QueryInput) {
			stop := startTimer(q.k)
			res, err := q.f(ctx, begin, end)
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
	begin, err := time.Parse("20060102", r.FormValue("begin"))
	if err != nil {
		log.Println("Error parsing begin time.", err.Error())
		cancel()
	}
	end, err := time.Parse("20060102", r.FormValue("end"))
	if err != nil {
		log.Println("Error parsing end time.", err.Error())
		cancel()
	}

	result := make(ResultList)
	var mux sync.Mutex
	var wg sync.WaitGroup
	// Process queries
	for k, f := range queries {
		wg.Add(1)
		go func(
			ctx context.Context,
			begin, end time.Time,
			k string,
			f QueryFunction,
		) {
			defer wg.Done()
			defer startTimer(k)()

			res, err := f(ctx, begin, end)
			if err != nil {
				log.Printf(
					"Query %s failed with error %s\n",
					k,
					err.Error(),
				)
			}
			// Gather results
			for i, w := range res {
				mux.Lock()
				result[i] = w
				mux.Unlock()
			}
		}(ctx, begin, end, k, f)
	}
	// Display results
	wg.Wait()
	_, err = io.WriteString(w, fmt.Sprintf("%# v", result))
	if err != nil {
		log.Println(err)
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
	queries = make(QueryList)
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
