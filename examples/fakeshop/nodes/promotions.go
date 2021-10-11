package nodes

import (
	"context"
	"fmt"

	"github.com/heetch/universe/src/services/pickup-experience/core/internal/composer"
)

func PromotionsBuilder(customerID string) composer.NodeBuilderFn {
	return func(c *context.Context, df DataFetchder) (interface{}, error) {
		return nil, fmt.Errorf("no implemented")
	}
}
