package daterange

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

type key int

// DateRangeKey the key
const DateRangeKey key = 0

// DateRange represents a range between 2 dates
type DateRange struct {
	Begin, End time.Time
}

// NewContext creates a context containing a DateRange
func NewContext(ctx context.Context, date DateRange) context.Context {
	return context.WithValue(ctx, DateRangeKey, date)
}

// FromContext extracts a DateRange from a context
func FromContext(ctx context.Context) (DateRange, bool) {
	// type assertion, ok=false if assertion fails
	t, ok := ctx.Value(DateRangeKey).(DateRange)
	return t, ok
}

// FromRequest Create a DateRange from Request
func FromRequest(r *http.Request) (date DateRange, err error) {
	v := struct {
		Begin int64
		End   int64
	}{}
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Println(err)
		return
	}
	err = json.Unmarshal(body, &v)
	if err != nil {
		log.Println(err)
		return
	}

	begin := time.Unix(v.Begin, 0)
	end := time.Unix(v.End, 0)

	date = DateRange{begin, end}
	return
}

// NewContextFromRequest creates a Context containing the DateRange
// specified in the Request
func NewContextFromRequest(
	ctx context.Context,
	r *http.Request,
) (
	res context.Context,
	err error,
) {
	date, err := FromRequest(r)
	if err != nil {
		return
	}
	res = NewContext(ctx, date)
	return
}
