package query

import (
	"context"
	"io"

	"github.com/influxdata/flux"
	"github.com/influxdata/flux/csv"
	platform "github.com/influxdata/influxdb"
	"github.com/influxdata/influxdb/kit/tracing"
)

// QueryServiceBridge implements the QueryService interface while consuming the AsyncQueryService interface.
type QueryServiceBridge struct {
	AsyncQueryService AsyncQueryService
}

func (b QueryServiceBridge) Query(ctx context.Context, req *Request) (flux.ResultIterator, error) {
	query, err := b.AsyncQueryService.Query(ctx, req)
	if err != nil {
		return nil, err
	}
	return flux.NewResultIteratorFromQuery(query), nil
}

// QueryServiceProxyBridge implements QueryService while consuming a ProxyQueryService interface.
type QueryServiceProxyBridge struct {
	ProxyQueryService ProxyQueryService
}

func (b QueryServiceProxyBridge) Query(ctx context.Context, req *Request) (flux.ResultIterator, error) {
	d := csv.Dialect{ResultEncoderConfig: csv.DefaultEncoderConfig()}
	preq := &ProxyRequest{
		Request: *req,
		Dialect: d,
	}

	r, w := io.Pipe()
	asri := &asyncStatsResultIterator{statsReady: make(chan struct{})}

	go func() {
		stats, err := b.ProxyQueryService.Query(ctx, w, preq)
		_ = w.CloseWithError(err)
		asri.stats = stats
		close(asri.statsReady)
	}()

	dec := csv.NewMultiResultDecoder(csv.ResultDecoderConfig{})
	ri, err := dec.Decode(r)
	asri.ResultIterator = ri
	return asri, err
}

type asyncStatsResultIterator struct {
	flux.ResultIterator

	// Channel that is closed when stats have been written.
	statsReady chan struct{}

	// Statistics gathered from calling the proxy query service.
	// This field must not be read until statsReady is closed.
	stats flux.Statistics
}

func (i *asyncStatsResultIterator) Release() {
	i.ResultIterator.Release()
}

func (i *asyncStatsResultIterator) Statistics() flux.Statistics {
	<-i.statsReady
	return i.stats
}

// ProxyQueryServiceAsyncBridge implements ProxyQueryService while consuming an AsyncQueryService
type ProxyQueryServiceAsyncBridge struct {
	AsyncQueryService AsyncQueryService
}

func (b ProxyQueryServiceAsyncBridge) Query(ctx context.Context, w io.Writer, req *ProxyRequest) (flux.Statistics, error) {
	span, ctx := tracing.StartSpanFromContext(ctx)
	defer span.Finish()

	q, err := b.AsyncQueryService.Query(ctx, &req.Request)
	if err != nil {
		return flux.Statistics{}, tracing.LogError(span, err)
	}

	results := flux.NewResultIteratorFromQuery(q)
	defer results.Release()

	encoder := req.Dialect.Encoder()
	if _, err := encoder.Encode(w, results); err != nil {
		return flux.Statistics{}, tracing.LogError(span, err)
	}
	results.Release()

	stats := results.Statistics()
	return stats, nil
}

// REPLQuerier implements the repl.Querier interface while consuming a QueryService
type REPLQuerier struct {
	// Authorization is the authorization to provide for all requests
	Authorization *platform.Authorization
	// OrganizationID is the ID to provide for all requests
	OrganizationID platform.ID
	QueryService   QueryService
}

func (q *REPLQuerier) Query(ctx context.Context, compiler flux.Compiler) (flux.ResultIterator, error) {
	req := &Request{
		Authorization:  q.Authorization,
		OrganizationID: q.OrganizationID,
		Compiler:       compiler,
	}
	return q.QueryService.Query(ctx, req)
}
