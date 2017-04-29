package fakeprovider

import (
	"context"
	"time"

	"github.com/miguelbernadi/dashboard/provider"
)

// FakeProvider is a fake data provider to test the server
type FakeProvider struct{}

// Login returns nil
func (p FakeProvider) Login() error {
	return nil
}

// Register returns function map with bogus data
func (p FakeProvider) Register() (provider.QueryList, error) {
	funcs := provider.QueryList{
		"Name": func(
			ctx context.Context,
			date1, date2 time.Time,
		) (
			provider.ResultList,
			error,
		) {
			result := provider.ResultList{
				"Name": "Hydra",
			}
			return result, nil
		},
		"Heads": func(
			ctx context.Context,
			date1, date2 time.Time,
		) (
			provider.ResultList,
			error,
		) {
			result := provider.ResultList{
				"Heads": 8,
			}
			return result, nil
		},
	}
	return funcs, nil
}
