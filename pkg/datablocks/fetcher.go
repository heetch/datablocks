package datablocks

import (
	"context"
	"fmt"
	"sync"
)

type DataFetcherFn func(c context.Context) (interface{}, error)

// AsyncFecthData holds the result for fetching some data
type AsyncFetchData struct {
	Hash   string
	Result interface{}
	Err    error
}

// AsyncFechReq provides a function to fetch some data, and
// a hash identifying what we want to fetch.
type AsyncFetchReq struct {
	Fetcher DataFetcherFn
	Hash    string
}

// TODO: we need to convert this interface to a function type definition
type DataFetcher interface {
	Fetch(ctx context.Context, dataReq *AsyncFetchReq) (<-chan *AsyncFetchData, error)
	WaitForFetch(ctx context.Context, dataReq *AsyncFetchReq) (*AsyncFetchData, error)
	WaitForFetches(ctx context.Context,
		dataReqs ...*AsyncFetchReq) ([]*AsyncFetchData, error)
}

// cachedAsyncData holds the on flight or already received
// data from a fetching function.
// When the request is on flight `fetchedData` will be still null
type cachedAsyncData struct {
	notifyChan  chan *AsyncFetchData
	fetchedData *AsyncFetchData

	// in pendingNotification we put the count of times
	// we need to send the event through the channel once
	// the in-flight data fetch is finished
	numPendingNotifications int
}

// DataFetcherImpl is an implementation of a DataFetcher with in-memory cache
// (used for a single request)
type DataFetcherImpl struct {
	dataMut     sync.Mutex
	data        map[string]*cachedAsyncData
	dataChanCap int
}

// NewDataFetcherImpl returns a DataFetcher implementation
// that notifies clients of the API that the result is ready.
//
// dataChanCap defines the capacity for al the chans created
// to notify the clients of the API.
// In the case of response builder, it can be the number of
// nodes, as we do not expect a node to request the same data
// twice (with the same params)
func NewDataFetcherImpl(dataChanCap int) *DataFetcherImpl {
	if dataChanCap <= 1 {
		dataChanCap = 16
	}
	return &DataFetcherImpl{
		data:        make(map[string]*cachedAsyncData, dataChanCap),
		dataChanCap: dataChanCap,
	}
}

func (df *DataFetcherImpl) Fetch(ctx context.Context, req *AsyncFetchReq) (<-chan *AsyncFetchData, error) {
	if req.Fetcher == nil {
		return nil, fmt.Errorf("null datafetcher")
	}

	if len(req.Hash) == 0 {
		// TODO: if there is no hash provided we can:
		// - return an error
		// - compute a default one based on the function pointer
		req.Hash = fmt.Sprintf("AnonymousData_%v", &req.Fetcher)
	}

	var err error
	df.dataMut.Lock()
	cad, reqExists := df.data[req.Hash]
	if reqExists {
		if cad.numPendingNotifications > 0 {
			// fetching on flight
			cad.numPendingNotifications += 1
		} else {
			// the fetching has finished: if chan would block we launch
			// a goroutine to block on notification.
			select {
			case cad.notifyChan <- cad.fetchedData:
			default:
				go func() { cad.notifyChan <- cad.fetchedData }()
			}
		}
	} else {
		// there is no previous request to fetch this data
		cad = &cachedAsyncData{
			notifyChan:              make(chan *AsyncFetchData, df.dataChanCap),
			numPendingNotifications: 1,
		}
		df.data[req.Hash] = cad
	}
	df.dataMut.Unlock()

	// if the request was new, we launch a backgroundFetch
	if !reqExists {
		go df.backgroundFetch(ctx, cad, req)
	}
	return cad.notifyChan, err
}

// backgroundFetch runs in the background to fetch some data
func (df *DataFetcherImpl) backgroundFetch(ctx context.Context,
	cad *cachedAsyncData, req *AsyncFetchReq) {

	res := &AsyncFetchData{
		Hash: req.Hash,
	}
	res.Result, res.Err = req.Fetcher(ctx)

	// we need to lock the results
	df.dataMut.Lock()
	if _, ok := df.data[req.Hash]; !ok {
		// TODO: Log this!: if this happens we have a big bug somewhere
		df.dataMut.Unlock()
		return
	}
	// we will notify clients after unlock:
	numNotifications := cad.numPendingNotifications
	cad.numPendingNotifications = 0
	cad.fetchedData = res
	df.data[res.Hash] = cad
	df.dataMut.Unlock()

	// at this point we do not care if we are blocking on the channel as the
	// data is already in place, and this function is already running in a
	// goroutine
	for ; numNotifications > 0; numNotifications-- {
		cad.notifyChan <- res
	}
}

// WaitForFetch is a helper function to launch a parallel request
// and immediatly wait for its response
func (df *DataFetcherImpl) WaitForFetch(ctx context.Context,
	req *AsyncFetchReq) (*AsyncFetchData, error) {
	c, err := df.Fetch(ctx, req)
	if err != nil {
		return nil, err
	}
	select {
	case res := <-c:
		return res, nil
	case <-ctx.Done():
		return nil, fmt.Errorf("ctx cancelled")
	}
}

// WaitForFetches is a helper function to launch several parallel requests
// and immediatly wait for all of them.
// The list of results follow the same order than the reqs params.
func (df *DataFetcherImpl) WaitForFetches(ctx context.Context,
	reqs ...*AsyncFetchReq) ([]*AsyncFetchData, error) {

	cs := make([]<-chan *AsyncFetchData, len(reqs))
	var err error
	for idx, req := range reqs {
		cs[idx], err = df.Fetch(ctx, req)
		if err != nil {
			return []*AsyncFetchData{}, err
		}
	}

	data := make([]*AsyncFetchData, len(reqs))
	// listen for all the chans in the same order than the list
	for idx, c := range cs {
		select {
		case res := <-c:
			data[idx] = res
		case <-ctx.Done():
			return []*AsyncFetchData{}, fmt.Errorf("ctx cancelled")
		}
	}
	return data, nil
}
