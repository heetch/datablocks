package nodes

import (
	"context"
	"fmt"

	"github.com/heetch/universe/src/services/pickup-experience/core/internal/composer"
	"github.com/heetch/universe/src/services/pickup-experience/core/internal/domain/phasingmoder/fetchers"
)

// CustomerNodeBuilder returns a function of type `NodeBuilderFn`
// making a closure with the input params
func CustomerNodeBuilder(customerID string) composer.NodeBuilderFn {
	return func(ctx context.Context, df composer.DataFetcher) (interface{}, error) {
		// this is an example of not waiting immediately for the result
		// of a fetch rfequest
		c, err := df.Fetch(fetchers.CustomerAsyncReq(customerID))

		if err != nil {
			return nil, err
		}

		// Here we could keep doing some other stuff

		// We want the node builders to be cancellable:
		select {
		case res := <-orderChan:
			// here we need to cast the Result to the proper type to work
			// with it
			return res.Result, res.Err
		case <-ctx.Done():
			return nil, fmt.Errorf("ctx cancelled")
		}
	}
}
