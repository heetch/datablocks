package fetchers

import (
	"context"
	"fmt"
	"strings"

	"github.com/heetch/datablocks/pkg/datablocks"
)

// PromotionsAsynReq prepares a function that conforms to the
// `DataFetcherFn` data type by preparing a closure on the
// input params. It also prepares a "hash" or unique ID for
// this function call (and params) so the datafecher can
// maintain a local temporary cache for the result.
func PromotionsAsyncReq(customerID string) *datablocks.AsyncFetchReq {
	// we have to build our unique ID, based on the input params
	var hashBuilder strings.Builder
	hashBuilder.Grow(len(customerID) + len("Promotions_"))
	hashBuilder.WriteString("Promotions_")
	hashBuilder.WriteString(customerID)

	ad := datablocks.AsyncFetchReq{
		Fetcher: func(c context.Context) (interface{}, error) {
			return nil, nil
		},
		Hash: hashBuilder.String(),
	}
	return &ad
}

// Promotion contains the description of an special offer
// and its product ID
type Promotion struct {
	Description string
	ProductID   string
}

// ToPromotions is a helper function to cast the interface{}
// result from fetching the data to an Promotions type
func ToPromotions(d *datablocks.AsyncFetchData) (*Promotions, error) {
	if d == nil {
		return nil, fmt.Errorf("cannot cast nil value")
	}
	out, ok := d.Result.(*Promotions)
	if !ok {
		return nil, fmt.Errorf("cannot cast")
	}
	return out, nil
}
