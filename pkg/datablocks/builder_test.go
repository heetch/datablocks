package datablocks

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// newTestDelayedNodeBuilder fakes the time it will take to build a node,
// and returns a predefined response, or error provided in err if is not nil
func newTestDelayedNodeBuilder(millis int, err error) NodeBuilderFn {
	return func(ctx context.Context, df DataFetcher) (interface{}, error) {
		timer := time.After(time.Duration(millis) * time.Millisecond)
		res := map[string]string{
			"delay":   fmt.Sprintf("%d", millis),
			"started": fmt.Sprintf("%v", time.Now().UTC()),
		}
		select {
		case <-timer:
		case <-ctx.Done():
			return nil, fmt.Errorf("cancelled")
		}
		if err != nil {
			return nil, err
		}
		return res, nil
	}
}

func Test_BuilderWithNoNodes(t *testing.T) {
	nodesConf := []NodeConf{}
	storageKey := "test_response"
	storage := NewNopKeyValStorage()
	// we "hint" the data fetcher to use the number of nodes as the
	// maximum number of buffer for the chan responses, as we assume
	// no NodeBuilder will try to fetch twice the same data
	dataFetcher := NewDataFetcherImpl(len(nodesConf))

	rb := NewResponseBuilder(storageKey, storage, dataFetcher, nodesConf, 800)

	reqReady := make(chan bool, 1)
	fullReady := make(chan bool, 1)

	rb.Build(context.Background(), reqReady, fullReady)

	timeout := time.After(time.Second)
	select {
	case readyRes := <-reqReady:
		if !readyRes {
			t.Errorf("readyRes, want true, got false")
		}
	case <-timeout:
		t.Errorf("time expired")
		return
	}

	select {
	case fullRes := <-fullReady:
		if !fullRes {
			t.Errorf("fullRes, want true, got false")
		}
	case <-timeout:
		t.Errorf("time expired")
		return
	}
}

func Test_BuilderWithSingleStaticNode(t *testing.T) {

	nodesConf := []NodeConf{
		NodeConf{
			Key:      "foo",
			Static:   true,
			Required: true,
			Builder:  newTestDelayedNodeBuilder(5, nil),
		},
	}

	storage := NewNopKeyValStorage()
	dataFetcher := NewDataFetcherImpl(len(nodesConf))
	rb := NewResponseBuilder("test_response", storage, dataFetcher, nodesConf, 800)
	reqReady := make(chan bool, 1)
	fullReady := make(chan bool, 1)

	rb.Build(context.Background(), reqReady, fullReady)

	timeout := time.After(time.Millisecond * time.Duration(1000))
	select {
	case readyRes := <-reqReady:
		if !readyRes {
			t.Errorf("readyRes, want true, got false")
			return
		}
	case <-timeout:
		t.Errorf("time expired")
		return
	}

	output := rb.Result()
	if len(output) != 1 {
		t.Errorf("output len, want 1, got %d", len(output))
		return
	}

	fooVal, ok := output["foo"].(map[string]string)
	if !ok {
		t.Errorf("want a map for foo value")
		return
	}

	if res, ok := fooVal["delay"]; res != "5" {
		t.Errorf("duration, want 5, got %v (%v) -> %#v\n", res, ok, fooVal)
		return
	}
}

func Test_BuilderWithSeveralStaticNodesOneTimingOut(t *testing.T) {
	// this test contains two required nodes that should complete in 5 ms,
	// and an optional node that takes 500 ms and should not complete because
	// we thell the response builder to finish the builing nodes in 50 ms
	// and optional node that should complete in 10 ms,
	nodesConf := []NodeConf{
		NodeConf{
			Key:      "n0",
			Static:   true,
			Required: true,
			Builder:  newTestDelayedNodeBuilder(5, nil),
		},
		NodeConf{
			Key:      "n1",
			Static:   true,
			Required: true,
			Builder:  newTestDelayedNodeBuilder(3, nil),
		},
		NodeConf{
			Key:      "n2",
			Static:   true,
			Required: false,
			Builder:  newTestDelayedNodeBuilder(500, nil),
		},
		NodeConf{
			Key:      "n3",
			Static:   true,
			Required: false,
			Builder:  newTestDelayedNodeBuilder(10, nil),
		},
	}

	storage := NewNopKeyValStorage()
	dataFetcher := NewDataFetcherImpl(len(nodesConf))
	rb := NewResponseBuilder("test_response", storage, dataFetcher, nodesConf, 50)
	reqReady := make(chan bool, 1)
	fullReady := make(chan bool, 1)

	rb.Build(context.Background(), reqReady, fullReady)

	timeout := time.After(time.Millisecond * time.Duration(1000))
	select {
	case readyRes := <-reqReady:
		if !readyRes {
			t.Errorf("readyRes, want true, got false")
			return
		}
	case <-timeout:
		t.Errorf("time expired")
		return
	}

	output := rb.Result()
	if len(output) != 2 {
		t.Errorf("output len, want 2, got %d", len(output))
		return
	}

	// now we wait for some optionals to complete:
	select {
	case fullRes := <-fullReady:
		if fullRes {
			// one of the optional will timeout, so it must be missing
			t.Errorf("fullRes should be false")
			return
		}
	case <-timeout:
		t.Errorf("time expired waiting for full Ready")
		return
	}

	output = rb.Result()
	if len(output) != 3 {
		t.Errorf("output len, want 3, got %d", len(output))
		return
	}

	_, ok := output["n2"]
	if ok {
		t.Errorf("node n2 should be missing")
		return
	}
	t.Errorf("fail")
}
