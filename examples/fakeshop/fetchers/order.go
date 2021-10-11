package fetchers

import (
	"context"
	"fmt"

	"github.com/heetch/universe/src/services/pickup-experience/core/internal/composer"
)

// Order contains all the products for an order
type Order struct {
	OrderID    string
	ProductIDs []string
}

// OrderAsynReq prepares a function that conforms to the
// `DataFetcherFn` data type by preparing a closure on the
// input params. It also prepares a "hash" or unique ID for
// this function call (and params) so the datafecher can
// maintain a local temporary cache for the result.
func OrderAsyncReq(customerID string) *composer.AsyncFetchReq {
	// we have to build our unique ID, based on the input params
	ad := composer.AsyncFetchReq{
		Fetcher: func(c context.Context) (interface{}, error) {
			return &Order{
				OrderID:    "order_1",
				ProductIDs: []string{},
			}, nil
		},
		Hash: fmt.Sprintf("Order_%s", customerID),
	}
	return &ad
}

// ToOrder is a helper function to cast the interface{}
// result from fetching the data to an Order type
func ToOrder(d *composer.AsyncFetchData) (*Order, error) {
	if d == nil {
		return nil, fmt.Errorf("cannot cast nil value")
	}
	out, ok := d.Result.(*Order)
	if !ok {
		return nil, fmt.Errorf("cannot cast")
	}
	return out, nil
}
