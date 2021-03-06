package sockjs

/*
Cotains package internal types (not public)
*/

import (
	"errors"
	"io"
	"time"
)

// Error variable
var ErrConnectionClosed = errors.New("Connection closed.")

type context struct {
	Config
	HandlerFunc
	connections
}

type conn struct {
	context
	input_channel    chan []byte
	output_channel   chan []byte
	timeout          time.Duration
	httpTransactions chan *httpTransaction
}

func newConn(ctx *context) *conn {
	return &conn{
		input_channel:    make(chan []byte),
		output_channel:   make(chan []byte, 64),
		httpTransactions: make(chan *httpTransaction),
		timeout:          time.Second * 30,
		context:          *ctx,
	}
}

func (c *conn) ReadMessage() ([]byte, error) {
	if val, ok := <-c.input_channel; ok {
		return val[1 : len(val)-1], nil
	}
	return []byte{}, io.EOF
}

func (c *conn) WriteMessage(val []byte) (count int, err error) {
	val2 := make([]byte, len(val))
	copy(val2, val)
	select {
	case c.output_channel <- val2:
	case <-time.After(c.timeout):
		return 0, ErrConnectionClosed
	}
	return len(val), nil
}

func (c *conn) Close() (err error) {
	defer func() {
		if recover() != nil {
			err = ErrConnectionClosed
		}
	}()
	close(c.input_channel)
	close(c.output_channel)
	return
}

type connectionStateFn func(*conn) connectionStateFn

func (c *conn) run(cleanupFn func()) {
	for state := openConnectionState; state != nil; {
		state = state(c)
	}
	c.Close()
	cleanupFn()
}
