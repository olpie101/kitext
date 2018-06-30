package nats

import (
	"context"
)

// RequestFunc may take information from an NATS request and put it into a
// request context. In Servers, RequestFuncs are executed prior to invoking the
// endpoint. In Clients, RequestFuncs are executed after creating the request
// but prior to invoking the client.
type RequestFunc func(context.Context, *Msg) context.Context

// ClientRequestFunc may take information from context and use it to construct
// metadata headers to be transported to the server. ClientRequestFuncs are
// executed after creating the request but prior to sending the NATS Msg to
// the server.
//type ClientRequestFunc func(context.Context, *Msg) context.Context

// ServerResponseFunc may take information from a request context and use it to
// manipulate a Msg. ServerResponseFuncs are only executed in
// servers, after invoking the endpoint but prior to writing a response.
type ServerResponseFunc func(context.Context, *Msg) context.Context

// ClientResponseFunc may take information from an NATS Msg and make the
// response available for consumption. ClientResponseFuncs are only executed in
// clients, after a request has been made, but prior to it being decoded.
type ClientResponseFunc func(context.Context, *Msg) context.Context

// SetResponseHeader returns a ServerResponseFunc that sets the given header.
func SetResponseHeader(key, val string) ServerResponseFunc {
	return func(ctx context.Context, m *Msg) context.Context {
		m.Header().Set(key, val)
		return ctx
	}
}

// SetRequestHeader returns a RequestFunc that sets the given header.
func SetRequestHeader(key, val string) RequestFunc {
	return func(ctx context.Context, m *Msg) context.Context {
		m.Header().Set(key, val)
		return ctx
	}
}
