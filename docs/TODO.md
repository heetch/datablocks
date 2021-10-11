- Remove `DataFetcherImpl` name that evokes Java code
- Hide `Hash` from the `AsyncFetchReq`
- Provide a simpler build approach:

> All this logic should go inside the node builder. Levering complexity to the caller requires extra effort, duplication of code and it's error prone. APIs should be as simple as possible.
> I'd make the assumption the Builder will return the result given a set of time defined in the context.
> Is spread that around all the API where it can be done in a single place.

```
rb := NewResponseBuilder(storage, conf)
res, err := rb.Build(ctx)
The conf will contain the timeout for building nodes, the relationship between fetchers and the output, the storageKey is another configuration parameter. This case I see the Conf object pattern much better suited.
```

- Use `sync.Once` to make sure the Builder's `build` function is only called once.
- Remove `WaitForFetch` as the same result can be obtained with the more generig `WaitForFetches` that fetches all the requests.
