package datablocks

import (
	"context"
)

// NodeBuilderFn defines the interface required for a
type NodeBuilderFn func(ctx context.Context, df DataFetcher) (interface{}, error)

// NodeConf has the information about how to build one
// of the entries of the final result
type NodeConf struct {
	Key      string
	Static   bool
	Required bool

	Builder NodeBuilderFn
}
