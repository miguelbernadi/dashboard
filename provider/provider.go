package provider

import (
	"context"
	"time"
)

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

// Provider must be satisfied by data providers
type Provider interface {
	Login() error
	Register() (QueryList, error)
}
