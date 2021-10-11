package nodes

import (
	"context"

	"github.com/heetch/universe/src/services/pickup-experience/core/internal/composer"
	"github.com/heetch/universe/src/services/pickup-experience/core/internal/domain/phasingmoder/fetchers"
)

// OrderNodeBuilder returns a function of type `NodeBuilderFn`
// making a closure with the input params
func OrderNodeBuilder(customerID string) composer.NodeBuilderFn {
	return func(ctx context.Context, df composer.DataFetcher) (interface{}, error) {
		res, err := df.WaitForFetch(fetchers.OrderAsyncReq(orderID))
		if err != nil {
			return nil, err
		}
		return res.Result, res.Err
	}
}
