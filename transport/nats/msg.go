package nats

import (
	"strings"

	"fmt"

	"google.golang.org/grpc/codes"
)

type Encoding string

type headerKey string

const (
	ENCODING headerKey = "enc"
)

const (
	ENCODING_JSON   = "json"
	ENCODING_PROTO3 = "proto3"

	headerNatsSubject      = "nats-subject"
	headerNatsReplySubject = "nats-reply-subject"
)

type Headers map[string]string

type Error interface {
	error

	// Returns the short phrase depicting the classification of the error.
	Code() uint32

	// Returns the error details message.
	Message() string
}

func (m *Msg) Header() Headers {
	return m.Headers
}

func (m *Msg) Subject() string {
	return m.Headers[headerNatsSubject]
}

func (m *Msg) Reply() string {
	return m.Headers[headerNatsReplySubject]
}

//func (m *Msg) Encoding() Encoding {
//	m.Headers[]
//}

func (m *Msg) WithEncoding(enc Encoding) {
	m.Headers[string(ENCODING)] = string(enc)
}

// Set sets the header entries associated with key to
// the single element value. It replaces any existing
// values associated with key.
func (h Headers) Set(key, value string) {
	key = strings.ToLower(key)
	h[key] = value
}

// Get gets the first value associated with the given key.
// It is case insensitive;
// If there are no values associated with the key, Get returns "".
func (h Headers) Get(key string) string {
	return h[strings.ToLower(key)]
}

// Del deletes the values associated with key.
func (h Headers) Del(key string) {
	delete(h, strings.ToLower(key))
}

func (s *Status) Err() error {
	if s.Code != uint32(codes.OK) {
		return transError{
			c: s.Code,
			m: s.Message,
		}
	}
}

func WrapError(err error) *Status {
	if err == nil {
		return nil
	}
	e := &Status{}

	er, ok := err.(Error)

	if !ok {
		e.Code = uint32(codes.Unknown)
		e.Message = err.Error()
		return e
	}

	e.Code = er.Code()
	e.Message = er.Message()

	return e
}

type transError struct {
	c uint32
	m string
}

func (e transError) Error() string {
	return fmt.Sprintf("(%d) %s", e.c, e.m)
}

func (e transError) Code() uint32 {
	return e.c
}

func (e transError) Message() string {
	return e.m
}
