package nodes

import (
	"context"
	"fmt"

	"github.com/heetch/universe/src/services/pickup-experience/core/internal/composer"
	"github.com/heetch/universe/src/services/pickup-experience/core/internal/domain/phasingmoder/fetchers"
)

// ProductsNodeBuilder returns a function of type `NodeBuilderFn`
// making a closure with the input params
func ProductsNodeBuilder(customerID string) composer.NodeBuilderFn {
	return func(ctx context.Context, df composer.DataFetcher) (interface{}, error) {
		f, err := df.WaitForFetch(ctx, fetchers.OrderAsyncReq(orderID))
		if err != nil {
			return nil, err
		}

		o, err := fetchers.ToOrder(f)
		if err != nil {
			return nil, err
		}

		f, err = df.WaitForFetch(fetchers.ProductAsyncReq(o.LegacyProductID))
		if err != nil {
			return nil, err
		}
		if f.Err != nil {
			return nil, fmt.Errorf("cannot fetch the data")
		}
		return f.Result, nil
	}
}
