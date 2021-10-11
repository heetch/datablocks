package datablocks

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// newTestAsyncFetchReq takes millis and fake param to build
// an AsyncFetchData (hash and a fetch function pair),
// that will return res and err result after millis Milliseconds
func newTestAsyncFetchReq(millis int, fakeParam string,
	res interface{}, err error) *AsyncFetchReq {
	hash := fmt.Sprintf("TestFn_%d_%s", millis, fakeParam)
	return &AsyncFetchReq{
		Fetcher: func(ctx context.Context) (interface{}, error) {
			timeout := time.After(time.Millisecond * time.Duration(millis))
			select {
			case <-timeout:
				return res, err
			case <-ctx.Done():
				return nil, fmt.Errorf("cancelled")
			}
		},
		Hash: hash,
	}
}

func Test_FetcherSimpleCase(t *testing.T) {
	df := NewDataFetcherImpl(10)

	testData := map[string]string{
		"a": "x",
		"b": "y",
	}

	ctx := context.Background()
	recv, err := df.Fetch(ctx, newTestAsyncFetchReq(1, "foo", testData, nil))
	if err != nil {
		t.Errorf("unexpected error %s", err.Error())
		return
	}

	timeout := time.After(time.Millisecond * time.Duration(10))
	var res interface{}
	select {
	case asyncData := <-recv:
		if asyncData.Err != nil {
			t.Errorf("unexpected error %s", asyncData.Err.Error())
			return
		}
		res = asyncData.Result
	case <-timeout:
		t.Errorf("time expired")
		return
	}

	receivedData, ok := res.(map[string]string)
	if !ok {
		t.Errorf("received data should be a dict")
		return
	}

	// actually, the map should be the very same pointer
	if receivedData["a"] != testData["a"] {
		t.Errorf("received data do not match")
		return
	}
}

func Test_FetcherCachedData(t *testing.T) {
	df := NewDataFetcherImpl(10)

	testData := map[string]string{
		"a": "x",
		"b": "y",
	}

	ctx := context.Background()
	recv, err := df.Fetch(ctx, newTestAsyncFetchReq(10, "foo", testData, nil))
	if err != nil {
		t.Errorf("unexpected error %s", err.Error())
		return
	}

	start := time.Now()
	var firstRunDuration, secondRunDuration time.Duration
	timeout := time.After(time.Millisecond * time.Duration(20))
	select {
	case <-recv:
		firstRunDuration = time.Since(start)
	case <-timeout:
		t.Errorf("time expired")
		return
	}

	if firstRunDuration < (time.Duration(10) * time.Millisecond) {
		t.Errorf("first run should take at least 10ms")
		return
	}
	start = time.Now()
	recv, _ = df.Fetch(ctx, newTestAsyncFetchReq(10, "foo", testData, nil))
	select {
	case <-recv:
		secondRunDuration = time.Since(start)
	case <-timeout:
		t.Errorf("time expired")
		return
	}

	if secondRunDuration > (time.Duration(1) * time.Millisecond) {
		t.Errorf("second run should be under 1ms: is already cached")
		return
	}
}
