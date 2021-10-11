package datablocks

import (
	"context"
	"encoding/json"
	"sync"
	"time"
)

const (
	DefaultBuildNodeTimeoutMillis int = 2000
)

// ResponseBuilder builds an response from the output
// of one or more node builders
type ResponseBuilder struct {
	storageKey string
	storage    KeyValStorage

	dataFetcher DataFetcher

	result      []NodeBuilderResult
	numRequired int

	numReqPending int // required fields that are remaining to fetch
	numReqErr     int // required nodes we could not fetch
	numOptPending int // optional fields that are remaining to fetch
	numOptErr     int // optional nodes we could not fetch

	requiredReady chan<- bool
	fullReady     chan<- bool

	lock sync.RWMutex
	// response map[string]interface{}

	buildNodeTimeoutMillis int

	// so we can keep stats of how long it took to build all the process
	buildStartTime time.Time
}

// NodeBuilderResults holds the output for a given NodeConf
//
// The `res` value could be nil and valid, so we need to know that it was
// actually the value fetched from the node.
type NodeBuilderResult struct {
	nodeConf NodeConf
	res      interface{}
	err      error
	fetched  bool
}

// NewReponseBuilder creates a node builder that can launch parallel
// 	node builders (see `NodeConf`).
//
// storageKey must be unique for a given (phase / state) set,
// 	and is up to the caller to build it
//
// buildNodeTimeoutMillis: is the maximum time it can take a node builder
// 	to return its response: the timeout is required to ensure that some bad
// 	behaved / stuck node builder causes the build process to not finish.
//
// Note:
// if we want to avoid loading dynamic nodes, because we just want
// to warm up the storage (i.e: when we receive a kafka event) the caller
// should take care of removing those nodes from `nodesConf`
func NewResponseBuilder(storageKey string, storage KeyValStorage,
	dataFetcher DataFetcher, nodesConf []NodeConf,
	buildNodeTimeoutMillis int) *ResponseBuilder {

	if buildNodeTimeoutMillis < 0 {
		buildNodeTimeoutMillis = DefaultBuildNodeTimeoutMillis
	}

	rb := &ResponseBuilder{
		storageKey:  storageKey,
		storage:     storage,
		dataFetcher: dataFetcher,
		result:      make([]NodeBuilderResult, len(nodesConf)),
		// numRequired is computed when we build nodesConf
		// numReqErr is 0 at the start
		// numOptPending is calculated at the end of this function
		// numOptErr is 0 at the start
		// requiredReady is passed as param when we launch the build step and can be null
		// fullReady is passed as param when we launc the build step and can be null
		// lock does not need initialization
		buildNodeTimeoutMillis: buildNodeTimeoutMillis,
		// buildStartTime is set at start time
	}

	exists := make(map[string]bool, len(nodesConf))

	rb.result = make([]NodeBuilderResult, 0, len(nodesConf))
	for _, n := range nodesConf {
		// sanity check that we do not have the same key twice (even it would
		// work by just overwritting the key)
		if _, ok := exists[n.Key]; ok {
			// TODO: log a warning of duplicate definition for a node
			continue
		} else {
			exists[n.Key] = true
		}

		rb.result = append(rb.result, NodeBuilderResult{
			nodeConf: NodeConf{
				Key:      n.Key,
				Static:   n.Static,
				Required: n.Required,
				Builder:  n.Builder,
			},
		})

		if n.Required {
			rb.numRequired += 1
		}
	}

	rb.numReqPending = rb.numRequired
	rb.numOptPending = len(rb.result) - rb.numRequired
	return rb
}

// Build launches a background goroutine that takes care of building the model
// and accepts an optional channel to signal when the required nodes are ready, and
// another channel to signal when building of the full response has finished.
//
// a value of true will be sent through requiredReady when all the required
//		nodes are build successfully
// a value of false will be sent through requiredReady as soon as one of the
//		required nodes failed to build
//
// WARNING: the returned data values should not be written at all, as that
//		data is going to be shared for reading when storing it in the storage.
//		Writing to data might cause data corruption and also migh cause panics.
// Key/Value pairs can be added to the returned map, as it is not shared (the map,
// not the other values stored).
func (rb *ResponseBuilder) Build(ctx context.Context, requiredReady chan<- bool,
	fullReady chan<- bool) {

	// we use the buildStartTime to know if we are already building the response
	rb.lock.Lock()
	if !rb.buildStartTime.IsZero() {
		rb.lock.Unlock()
		return
	}
	rb.buildStartTime = time.Now()
	rb.lock.Unlock()

	rb.requiredReady = requiredReady
	rb.fullReady = fullReady

	go rb.build(ctx)
}

// Result returns a new map with all the data fetched up to
// the moment the call is made. The map can be changed by the caller
// but not the values that it holds (those are shared with the ongoing
// build process).
func (rb *ResponseBuilder) Result() map[string]interface{} {
	rb.lock.RLock()
	defer rb.lock.RUnlock()

	// we convert the fetched nodes to a map
	m := make(map[string]interface{}, len(rb.result))
	for _, r := range rb.result {
		if r.fetched && r.err == nil {
			m[r.nodeConf.Key] = r.res
		}
	}
	return m
}

// build is the background process that computes the node state
func (rb *ResponseBuilder) build(ctx context.Context) {
	// we retrieve all static nodes, updating pending counters
	rb.fromStorage(ctx)

	if rb.numReqPending == 0 {
		noBlockChanBoolRes(rb.requiredReady, true)
		if rb.numOptPending == 0 {
			noBlockChanBoolRes(rb.fullReady, true)
			return
		}
	}

	finishedChan := make(chan *NodeBuilderResult, rb.numReqPending+rb.numOptPending)

	// node builders should respect context cancellation to limit response build time
	fetchCtx := ctx
	if rb.buildNodeTimeoutMillis > 0 {
		c, cancel := context.WithTimeout(fetchCtx,
			time.Millisecond*time.Duration(rb.buildNodeTimeoutMillis))
		fetchCtx = c
		defer cancel()
	}

	// launch the fetch of all parallel nodes
	for idx := range rb.result {
		go rb.buildNode(fetchCtx, &rb.result[idx], finishedChan)
	}

	// gather all results from parallel buildNode calls
	cancelled := false
	for (rb.numReqPending+rb.numOptPending) > 0 && !cancelled {
		select {
		case n := <-finishedChan:
			if n.nodeConf.Required {
				rb.numReqPending -= 1
			} else {
				rb.numOptPending -= 1
			}

			if n.err != nil {
				if n.nodeConf.Required {
					rb.numReqErr += 1
					if rb.numReqErr == 1 {
						noBlockChanBoolRes(rb.requiredReady, false)
					}
				} else {
					rb.numOptErr += 1
				}
			} else if n.nodeConf.Required && rb.numReqPending == 0 && rb.numReqErr == 0 {
				// we have all the required data
				noBlockChanBoolRes(rb.requiredReady, true)
			}
			// TODO: decide if we want to keep storing the partial result in storage
			// rb.toStorage(ctx)
		case <-ctx.Done():
			cancelled = true
		}
	}

	rb.toStorage(ctx)
	if rb.numReqPending == 0 && rb.numOptPending == 0 {
		noBlockChanBoolRes(rb.fullReady, rb.numOptErr == 0 && rb.numReqErr == 0)
	}
}

// noBlockChanBoolRes sends a boolean result to a chan without blocking
// We cannot wait for the client to read the channel as that would block
// the gathering of the results from nodes. Also we do not want to spawn
// a goroutine to notify the channel because we do not know if the client
// will listen to the channel. And notifyChan could also be a null channel
func noBlockChanBoolRes(notifyChan chan<- bool, value bool) {
	if notifyChan == nil {
		return
	}
	select {
	case notifyChan <- value:
	default:
	}
}

// buildNode build
func (rb *ResponseBuilder) buildNode(ctx context.Context, node *NodeBuilderResult,
	readyChan chan<- *NodeBuilderResult) {

	res, err := node.nodeConf.Builder(ctx, rb.dataFetcher)

	rb.lock.Lock()
	node.err = err
	node.res = res
	node.fetched = true
	rb.lock.Unlock()

	readyChan <- node
}

func (rb *ResponseBuilder) fromStorage(ctx context.Context) {
	// TODO: we might want to use []bytes in the storage interface
	// instead of string
	res, err := rb.storage.Get(ctx, rb.storageKey)
	if err != nil {
		// TODO: log the error
		return
	}

	if len(res) == 0 {
		// NO data found in storage
		return
	}

	resp := map[string]interface{}{}
	err = json.Unmarshal(res, &resp)
	if err != nil {
		// Bad data in Storage !? can that really happen ?
		// TODO: log the error
		return
	}

	for _, r := range rb.result {
		if val, ok := resp[r.nodeConf.Key]; ok {
			r.res = val
			r.fetched = true
			if r.nodeConf.Required {
				rb.numReqPending -= 1
			} else {
				rb.numOptPending -= 1
			}
		}
	}
}

func (rb *ResponseBuilder) toStorage(ctx context.Context) {
	staticNodes := make(map[string]interface{}, len(rb.result))

	rb.lock.RLock()
	for _, n := range rb.result {
		if n.nodeConf.Static && n.fetched {
			staticNodes[n.nodeConf.Key] = n.res
		}
	}
	rb.lock.RUnlock()

	b, err := json.Marshal(staticNodes)
	if err != nil {
		// TODO: log and continue
		return
	}

	err = rb.storage.Set(ctx, rb.storageKey, b)
	if err != nil {
		// TODO: log and continue,
		return
	}
}
