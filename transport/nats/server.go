package nats

import (
	"context"
	"fmt"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/nats-io/go-nats"
)

// Server wraps an endpoint and implements http.Handler.
type Server struct {
	e            endpoint.Endpoint
	addr         string
	dec          DecodeRequestFunc
	enc          EncodeResponseFunc
	before       []RequestFunc
	after        []ServerResponseFunc
	errEnc       ErrorEncoder
	finalizer    ServerFinalizerFunc
	logger       log.Logger
	transport    Transport
	subscription *nats.Subscription
}

func NewServe(
	e endpoint.Endpoint,
	addr string,
	dec DecodeRequestFunc,
	enc EncodeResponseFunc,
	errEnc ErrorEncoder,
	trans Transport,
	options ...ServerOption,
) (*Server, error) {
	s := &Server{
		e:         e,
		addr:      addr,
		dec:       dec,
		enc:       enc,
		errEnc:    errEnc,
		logger:    log.NewNopLogger(),
		transport: trans,
	}
	for _, option := range options {
		option(s)
	}

	//TODO add unsubscribe capabilities
	ctx := context.Background()
	fmt.Printf("subscribing to (%v)\n", addr)
	sub, err := s.transport.Subscribe(addr, s.handleMsg(ctx))
	if err != nil {
		return nil, err
	}
	s.subscription = sub
	return s, nil
}

func (s *Server) handleMsg(ctx context.Context) Handler {
	return func(inmsg *Msg) {
		fmt.Printf("tmsg %+v\n", inmsg)

		for _, f := range s.before {
			ctx = f(ctx, inmsg)
		}

		request, err := s.dec(ctx, inmsg)
		if err != nil {
			s.logger.Log("err", err, "msg", "unable to decode message")
			s.replyWithError(ctx, inmsg.Reply(), inmsg.Subject(), err)
			return
		}

		response, err := s.e(ctx, request)
		if err != nil {
			s.logger.Log("err", err, "msg", "unable to decode message")
			s.replyWithError(ctx, inmsg.Reply(), inmsg.Subject(), err)
			return
		}

		respMsg, err := s.enc(ctx, response)
		if err != nil {
			s.logger.Log("err", err, "msg", "unable to encode response")
			s.replyWithError(ctx, inmsg.Reply(), inmsg.Subject(), fmt.Errorf("%s: %v", "error encoding response:", err))
			return
		}

		fmt.Println("ae", respMsg)

		for _, f := range s.after {
			ctx = f(ctx, respMsg)
		}

		if len(inmsg.Reply()) == 0 {
			return
		}
		err = s.transport.Publish(ctx, inmsg.Reply(), respMsg)

		if err != nil {
			s.logger.Log("err", err, "msg", "error encoding response")
		}
	}
}

func (s *Server) replyWithError(ctx context.Context, replySub, origSub string, err error) {
	if len(replySub) == 0 {
		return
	}
	o := s.errEnc(ctx, fmt.Errorf("%s: %v", "unable to decode message:", err))
	if err = s.transport.Publish(ctx, replySub, o); err != nil {
		s.logger.Log("err", err, "msg", "unable publish reply", "sub", origSub, "reply-sub", replySub)
	}
}

//
// ServerOption sets an optional parameter for servers.
type ServerOption func(*Server)

// ServerBefore functions are executed on the HTTP request object before the
// request is decoded.
func ServerBefore(before ...RequestFunc) ServerOption {
	return func(s *Server) { s.before = append(s.before, before...) }
}

// ServerAfter functions are executed on the HTTP response writer after the
// endpoint is invoked, but before anything is written to the client.
func ServerAfter(after ...ServerResponseFunc) ServerOption {
	return func(s *Server) { s.after = append(s.after, after...) }
}

// ServerLogger is used to log non-terminal errors. By default, no errors
// are logged.
func ServerLogger(logger log.Logger) ServerOption {
	return func(s *Server) { s.logger = logger }
}

// ErrorEncoder is responsible for encoding an error to the ResponseWriter.
// Users are encouraged to use custom ErrorEncoders to encode HTTP errors to
// their clients, and will likely want to pass and check for their own error
// types. See the example shipping/handling service.
type ErrorEncoder func(ctx context.Context, err error) *Msg

// ServerFinalizerFunc can be used to perform work at the end of an HTTP
// request, after the response has been written to the client. The principal
// intended use is for request logging. In addition to the response code
// provided in the function signature, additional response parameters are
// provided in the context under keys with the ContextKeyResponse prefix.
type ServerFinalizerFunc func(ctx context.Context, code int)
