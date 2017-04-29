package main

import (
	"context"
	"time"
)

// FakeProvider is a fake data provider to test the server
type FakeProvider struct{}

// Login returns nil
func (p FakeProvider) Login() error {
	return nil
}

// Register returns function map with bogus data
func (p FakeProvider) Register() (
	map[string]func(
		ctx context.Context,
		date1, date2 time.Time,
	) (
		map[string]interface{}, error,
	),
	error) {

	funcs := map[string]func(
		ctx context.Context,
		date1, date2 time.Time,
	) (
		map[string]interface{}, error,
	){
		"Name": func(
			ctx context.Context,
			date1, date2 time.Time,
		) (
			map[string]interface{},
			error,
		) {
			result := map[string]interface{}{
				"Name": "Hydra",
			}
			return result, nil
		},
		"Heads": func(
			ctx context.Context,
			date1, date2 time.Time,
		) (
			map[string]interface{},
			error,
		) {
			result := map[string]interface{}{
				"Heads": 8,
			}
			return result, nil
		},
	}
	return funcs, nil
}
