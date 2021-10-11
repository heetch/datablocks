package fetchers

import (
	"context"
	"fmt"

	"github.com/heetch/datablocks/pkg/datablocks"
)

// Customer contains the profile information for a given customer
type Customer struct {
	Name        string
	Email       string
	MobilePhone string
}

// CustomerAsynReq prepares a function that conforms to the
// `DataFetcherFn` data type by preparing a closure on the
// input params. It also prepares a "hash" or unique ID for
// this function call (and params) so the datafecher can
// maintain a local temporary cache for the result.
func CustomerAsyncReq(customerID string) *datablocks.AsyncFetchReq {
	// we have to build our unique ID, based on the input params for the Hash value
	ad := datablocks.AsyncFetchReq{
		Fetcher: func(c context.Context) (interface{}, error) {
			return &Customer{
				Name:        "CustomerName",
				Email:       "customer@example.com",
				MobilePhone: "+44294012100",
			}, nil
		},
		Hash: fmt.Sprintf("Customer_%s", customerID),
	}
	return &ad
}

// ToCustomer is a helper function to cast the interface{}
// result from fetching the data to an Customer type
func ToCustomer(d *datablocks.AsyncFetchData) (*Customer, error) {
	if d == nil {
		return nil, fmt.Errorf("cannot cast nil value")
	}
	out, ok := d.Result.(*Customer)
	if !ok {
		return nil, fmt.Errorf("cannot cast")
	}
	return out, nil
}
