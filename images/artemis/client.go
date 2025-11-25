package artemis

import (
	"encoding/json"
	"fmt"
	"github.com/go-stomp/stomp/v3"
	"github.com/go-stomp/stomp/v3/frame"
	"log"
	"reflect"
)

type Client interface {
	// Publish publishes a message to a topic
	Publish(topicName string, message any, headers ...map[string]any) error
	// Send sends a message to a queue
	Send(queueName string, message any, headers ...map[string]any) error
	// Subscribe subscribes to topic
	//
	// returns a func to use to close/end the subscription
	Subscribe(topicName string, handler func(msg *stomp.Message)) (close func(), err error)
	// Consume consumes messages on a queue
	//
	// returns a func to use to close/end the consumer
	Consume(queueName string, handler func(msg *stomp.Message)) (close func(), err error)
	Close() error
}

func newClient(host string, options Options) (Client, error) {
	c := &client{
		marshaller: options.Marshaller,
	}
	var err error
	c.conn, err = stomp.Dial("tcp", host,
		stomp.ConnOpt.Login(options.username(), options.password()))
	return c, err
}

type client struct {
	conn       *stomp.Conn
	marshaller func(msg any) (body []byte, contentType string, err error)
}

func (c *client) Close() error {
	return c.conn.Disconnect()
}

func (c *client) Publish(topicName string, message any, headers ...map[string]any) (err error) {
	var body []byte
	var ct string
	if body, ct, err = c.encodeMessage(message); err == nil {
		err = c.conn.Send("/topic/"+topicName, ct, body, optsFromHeaders(headers)...)
	}
	return err
}

func (c *client) Send(queueName string, message any, headers ...map[string]any) (err error) {
	var body []byte
	var ct string
	if body, ct, err = c.encodeMessage(message); err == nil {
		err = c.conn.Send(queueName, ct, body, optsFromHeaders(headers)...)
	}
	return err
}

func optsFromHeaders(headers []map[string]any) []func(*frame.Frame) error {
	if len(headers) == 0 {
		return nil
	}
	m := make(map[string]string)
	for _, h := range headers {
		for k, v := range h {
			m[k] = fmt.Sprintf("%v", v)
		}
	}
	return []func(f *frame.Frame) error{
		func(f *frame.Frame) error {
			for k, v := range m {
				f.Header.Add(k, v)
			}
			return nil
		},
	}
}

func (c *client) Subscribe(topicName string, handler func(msg *stomp.Message)) (func(), error) {
	dest := "/topic/" + topicName
	sub, err := c.conn.Subscribe(dest, stomp.AckAuto)
	if err != nil {
		return nil, err
	}
	stop := make(chan struct{})
	go func() {
		defer func() {
			_ = sub.Unsubscribe()
		}()
		for {
			select {
			case <-stop:
				return
			case msg, ok := <-sub.C:
				if !ok {
					// subscription/channel closed...
					return
				}
				if msg.Err != nil {
					log.Printf("stomp subscribe message error on %q: %v", dest, msg.Err)
					return
				}
				handler(msg)
			}
		}
	}()
	return func() {
		close(stop)
	}, nil
}

func (c *client) Consume(queueName string, handler func(msg *stomp.Message)) (func(), error) {
	sub, err := c.conn.Subscribe(queueName, stomp.AckAuto)
	if err != nil {
		return nil, err
	}
	stop := make(chan struct{})
	go func() {
		defer func() {
			_ = sub.Unsubscribe()
		}()
		for {
			select {
			case <-stop:
				return
			case msg, ok := <-sub.C:
				if !ok {
					// subscription/channel closed...
					return
				}
				if msg.Err != nil {
					log.Printf("stomp consume message error on %q: %v", queueName, msg.Err)
					return
				}
				handler(msg)
			}
		}
	}()
	return func() {
		close(stop)
	}, nil
}

func (c *client) encodeMessage(msg any) ([]byte, string, error) {
	switch v := msg.(type) {
	case nil:
		return nil, "", nil
	case []byte:
		return v, "", nil
	case string:
		return []byte(v), "text/plain", nil
	default:
		if c.marshaller != nil {
			return c.marshaller(msg)
		}
		to := reflect.TypeOf(msg)
		if to.Kind() == reflect.Map || to.Kind() == reflect.Slice || to.Kind() == reflect.Struct {
			if data, err := json.Marshal(msg); err == nil {
				return data, "application/json", nil
			} else {
				return nil, "", err
			}
		} else {
			return []byte(fmt.Sprintf("%v", v)), "", nil
		}
	}
}
