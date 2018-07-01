package nats

import (
	"time"

	"sync"

	"fmt"

	"context"

	"github.com/go-kit/kit/log"
	nats "github.com/nats-io/go-nats"
	"google.golang.org/grpc/codes"
)

var (
	DefaultRequestTimeout = 2 * time.Second
)

// PublishOptions are options for a publication.
type PublishOptions struct {
	Cause string
}

type PublishOption func(*PublishOptions)

type RequestOptions struct {
	Cause   string
	Timeout time.Duration
}

type RequestOption func(*RequestOptions)

// RequestTimeout sets a request timeout duration.
func RequestTimeout(t time.Duration) RequestOption {
	return func(o *RequestOptions) {
		o.Timeout = t
	}
}

// RequestCause sets the cause of the request.
func RequestCause(s string) RequestOption {
	return func(o *RequestOptions) {
		o.Cause = s
	}
}

// SubscribeOptions are options for a subscriber.
type SubscribeOptions struct {
	Queue string
}

type SubscribeOption func(*SubscribeOptions)

// SubscribeQueue specifies the queue name of the subscriber.
func SubscribeQueue(q string) SubscribeOption {
	return func(o *SubscribeOptions) {
		o.Queue = q
	}
}

// Handler is the handler used by a subscriber. The return value may be nil if
// no output is yielded. If this is a request, the reply will be sent automatically
// with the reply value or an error if one occurred. If a reply is not expected
// and error occurs, it will be logged. The error can be inspected using status.FromError.
type Handler func(msg *Msg)

// Transport describes the interface
type Transport interface {
	// Publish publishes a message asynchronously to the specified subject.
	// The wrapped message is returned or an error. The error would only be due to
	// a connection issue, but does not reflect any consumer error.
	Publish(ctx context.Context, sub string, msg *Msg, opts ...PublishOption) error

	// Request publishes a message synchronously and waits for a response that
	// is decoded into the Protobuf message supplied. The wrapped message is
	// returned or an error. The error can inspected using status.FromError.
	Request(ctx context.Context, sub string, req *Msg, opts ...RequestOption) (*Msg, error)

	// Subscribe creates a subscription to a subject.
	Subscribe(sub string, hdl Handler, opts ...SubscribeOption) (*nats.Subscription, error)

	// Conn returns the underlying NATS connection.
	Conn() *nats.Conn

	// Close closes the transport connection and unsubscribes all subscribers.
	Close()

	// Set the logger.
	SetLogger(log.Logger)
}

// Connect is a convenience function establishing a connection with
// NATS and returning a transport.
func Connect(opts *nats.Options) (Transport, error) {
	conn, err := opts.Connect()
	if err != nil {
		return nil, err
	}

	logger := log.NewNopLogger()
	return &transport{
		logger: logger,
		conn:   conn,
	}, nil
}

// New returns a transport using an existing NATS connection.
func New(conn *nats.Conn) Transport {
	logger := log.NewNopLogger()

	return &transport{
		logger: logger,
		conn:   conn,
	}
}

type transport struct {
	logger log.Logger
	conn   *nats.Conn
	subs   []*nats.Subscription
	mux    sync.Mutex
}

func (c *transport) Publish(ctx context.Context, sub string, msg *Msg, opts ...PublishOption) error {
	data, err := msg.Marshal()

	if err != nil {
		return fmt.Errorf("unable to marshal message: %v", err)
	}
	if err := c.conn.Publish(sub, data); err != nil {
		return err
	}

	return nil
}

func (c *transport) Request(ctx context.Context, sub string, msg *Msg, opts ...RequestOption) (*Msg, error) {
	data, err := msg.Marshal()

	if err != nil {
		return nil, fmt.Errorf("unable to marshal message: %v", err)
	}

	nmsg, err := c.conn.RequestWithContext(ctx, sub, data)

	if err != nil {
		return nil, err
	}

	resp := &Msg{}
	err = resp.Unmarshal(nmsg.Data)

	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal message: %v", err)
	}

	return resp, nil
}

func (c *transport) Subscribe(sub string, hdl Handler, opts ...SubscribeOption) (*nats.Subscription, error) {
	s, err := c.conn.Subscribe(sub, msgHandler(c, hdl))

	if err != nil {
		return nil, fmt.Errorf("unable to subscribe to (%s): %v", sub, err)
	}

	c.mux.Lock()
	c.subs = append(c.subs, s)
	c.mux.Unlock()
	return s, nil

}

func (c *transport) SetLogger(l log.Logger) {
	c.logger = l
}

func (c *transport) Conn() *nats.Conn {
	return c.conn
}

func (c *transport) Close() {
	for _, sub := range c.subs {
		sub.Unsubscribe()
	}
	c.conn.Close()
}

func msgHandler(t *transport, hdl Handler) func(m *nats.Msg) {
	return func(m *nats.Msg) {
		ctx := context.Background()
		msg := &Msg{Headers: make(map[string]string)}

		err := msg.Unmarshal(m.Data)

		if err != nil {
			t.logger.Log(
				"error", err,
				"subject", m.Sub,
				"msg", "unable to unmarshal incomming message",
			)

			if len(m.Reply) > 0 {
				replyWithError(ctx, t, m.Sub.Subject, "error parsing request: %v", err)
			}
		}

		msg.Header()[headerNatsSubject] = m.Subject
		msg.Header()[headerNatsReplySubject] = m.Reply
		hdl(msg)
	}
}

func replyWithError(ctx context.Context, t *transport, sub string, format string, args ...interface{}) {
	msg := &Msg{
		Status: &Status{
			Code:    int32(codes.InvalidArgument),
			Message: fmt.Sprintf(format, args...),
		},
	}

	err := t.Publish(ctx, sub, msg)

	if err != nil {
		t.logger.Log(
			"error", err,
			"subject", sub,
			"msg", "unable to respond with error",
		)
	}
}
