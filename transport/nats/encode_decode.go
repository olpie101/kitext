package nats

import (
	"context"
)

// DecodeRequestFunc extracts a user-domain request object from a NATS
// Msg object. It's designed to be used with NATS Subscribers, for server-side
// endpoints. One straightforward DecodeRequestFunc could be something that
// JSON decodes from the message data to the concrete response type.
type DecodeRequestFunc func(context.Context, *Msg) (request interface{}, err error)

// EncodeRequestFunc encodes the passed request object into the NATS Msg
// object. It's designed to be used in NATS Publishers, for client-side
// endpoints. One straightforward EncodeRequestFunc could something that JSON
// encodes the object directly to the request body.
type EncodeRequestFunc func(context.Context, interface{}) (request *Msg, err error)

// EncodeResponseFunc encodes the passed response object to the NATS Msg.
// It's designed to be used in NATS Subscribers, for server-side
// endpoints. One straightforward EncodeResponseFunc could be something that
// JSON encodes the object directly to the response body.
type EncodeResponseFunc func(context.Context, interface{}) (response *Msg, err error)

// DecodeResponseFunc extracts a user-domain response object from an NATS
// Msg object. It's designed to be used in NATS Publishers, for client-side
// endpoints. One straightforward DecodeResponseFunc could be something that
// JSON decodes from the response body to the concrete response type.
type DecodeResponseFunc func(context.Context, *Msg) (response interface{}, err error)
