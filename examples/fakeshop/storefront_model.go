package phasingmodel

import (
	"context"
	"fmt"
	"time"

	"github.com/heetch/universe/src/services/pickup-experience/core/internal/datablocks"
	"github.com/heetch/universe/src/services/pickup-experience/core/internal/datablocks/nodes"
)

type FakeDependenciesThing struct {
	// here we need to put something regarding
	// how to pass dependencies for the fetching function
}

// GetStorefrontModel returns all the phasingmodel result
func GetStorefrontModel(ctx context.Context, userID string,
	deps *FakeDependenciesThing) (map[string]interface{}, error) {

	storageKey := fmt.Sprintf("ride.pickupxp.phasingmodel.%s", params.OrderID)
	// with the info or objects from deps, we create a redis keyval storage,
	// and from the input params w
	storage := datablocks.NewRedisKeyValStorage()

	// we "hint" the data fetcher to use the number of nodes as the
	// maximum number of buffer for the chan responses, as we assume
	// no NodeBuilder will try to fetch twice the same data
	dataFetcher := datablocks.NewDataFetcherImpl(len(nodesConf))

	// using the dependencies (grpc services to call, etc..), we build
	// the nodes and the fetcher functions:
	nodesConf := GetStorefrontNodesConf(deps)

	// we can setup a timeout for building the nodes, is not a global timeout
	// where we cancel all the response, is just a way to cut the time and
	// get all the nodes that could be fetched in that time (we should decide
	// if is good to maintain it or not)

	buildingNodesTimoutMs := 200
	rb := NewResponseBuilder(storageKey, storage, dataFetcher, nodesConf,
		buildingNodesTimoutMs)

	// we create a couple of buffered chans to receive the notifications
	// when the required nodes are ready (or are not going to be ready
	// because one already failed), and a notification when all the fetching
	// process has finished (that can be with all nodes ready, or only some
	// of them)
	reqReady := make(chan bool, 1)
	fullReady := make(chan bool, 1)

	// we launch the background ru
	rb.Build(context.Background(), reqReady, fullReady)

	// Attention here, because if we want to give some time to
	// dynamic and optional nodes to be built, we have to put
	// a timeout (because static nodes will be cached, and dynamic
	// ones will never be, so usually dynamic request will take longer
	// to complete)
	timeout := time.After(time.Millisecond * time.Duration(250))

	select {
	case readyRes := <-reqReady:
		if !readyRes {
			return nil, fmt.Errorf("failed to fetch required nodes")
		}
	case <-time.After(time.Millisecond * time.Duration(300)):
		// this time is different, because we want to give more
		// time for the required nodes to finish, than the optional
		return nil, t.Errorf("time expired")
	}

	// here we know that we allow some time for the optional stuff
	select {
	case <-fullReady:
		// at this point we do not care if all optional nodes are there
		// or not but we could have a metric for this
	case <-timeout:
		// in this case, we dont care if we could not finish
		return
	}

	// we get the response
	return rb.Result(), nil
}

// GetSorefrontNodesConf creates the configuration for the response for
// a storefront commerce.
func GetStorefrontNodesConf(customerID string, deps FakeDependenciesThing) ([]datablocks.NodeConf, err) {
	return []datablocks.NodeConf{
		datablocks.NodeConf{
			Key:      "customer",
			Static:   true,
			Required: true,
			Builder:  nodes.CustomerNodeBuilder(customerID),
		},
		datablocks.NodeConf{
			Key:      "order",
			Static:   true,
			Required: false,
			Builder:  nodes.OrderNodeBuilder(customerID),
		},
		datablocks.NodeConf{
			Key:      "suggestions",
			Static:   false,
			Required: false,
			Builder:  nodes.SuggestionsNodeBuilder(customerID),
		},
		datablocks.NodeConf{
			Key:      "products",
			Static:   true,
			Required: false,
			Builder:  nodes.ProductNodeBuilder(customerID),
		},
	}, nil
}
