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

func timeTrack(start time.Time, name string) {
	elapsed := time.Since(start)
	log.Printf("%s took %s", name, elapsed)
}

// ResultList is a list of query results
type ResultList map[string]interface{}

// QueryFunction is a function performing a query
type QueryFunction func(
	ctx context.Context, date1, date2 time.Time,
) (ResultList, error)

// QueryList represents a list of QueryFunctions
type QueryList map[string]QueryFunction

// Provider must be satisfied by data providers
type Provider interface {
	Login() error
	Register() (QueryList, error)
}

var providers = []Provider{
	FakeProvider{},
}

var queries QueryList

func query(w http.ResponseWriter, r *http.Request) {
	defer timeTrack(time.Now(), "search")
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
			log.Println("Starting", k)
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
			log.Println("Finished", k)
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

	log.Println("Starting server on port 8080")
	http.HandleFunc("/", welcome)
	http.HandleFunc("/search", query)

	err := server.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}
