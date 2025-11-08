package dragonfly

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-redis/redis/v7"
	"time"
)

type Client interface {
	// Publish publishes a message to a topic
	Publish(topicName string, message any) error
	// Send sends a message to a queue
	Send(queueName string, message any) error
	// Subscribe subscribes to topic
	//
	// returns a func to use to close/end the subscription
	Subscribe(topicName string, handler func(message string)) (close func())
	// Consume consumes messages on a queue
	//
	// returns a func to use to close/end the consumer
	Consume(queueName string, handler func(message string)) (close func())
	// Get retrieves a specific key
	Get(key string) (string, error)
	// Set stores a specific key value
	Set(key string, value string, expiry time.Duration) error
	// Exists checks whether a specific key exists
	Exists(key string) (bool, error)
	// Delete deletes a specific key (and returns whether it was deleted - i.e. existed)
	Delete(key string) (bool, error)
	// QueueLength returns the queue length for the given queue name
	QueueLength(queueName string) (int, error)
}

func newClient(host, port string) (Client, *redis.Client) {
	rc := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", host, port),
	})
	if err := rc.Ping().Err(); err != nil {
		return nil, nil
	}
	return &client{rc: rc}, rc
}

type client struct {
	rc *redis.Client
}

func (c *client) Publish(topicName string, message any) error {
	msg := ""
	switch mt := message.(type) {
	case string:
		msg = mt
	default:
		data, _ := json.Marshal(mt)
		msg = string(data)
	}
	return c.rc.Publish(topicName, msg).Err()
}

func (c *client) Send(queueName string, message any) error {
	msg := ""
	switch mt := message.(type) {
	case string:
		msg = mt
	default:
		data, _ := json.Marshal(mt)
		msg = string(data)
	}
	return c.rc.RPush(queueName, msg).Err()
}

func (c *client) Subscribe(topic string, handler func(message string)) func() {
	sub := c.rc.Subscribe(topic)
	if _, err := sub.Receive(); err != nil {
		_ = sub.Close()
		return func() {}
	}
	ch := sub.ChannelSize(256)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for msg := range ch {
			func() {
				defer func() { _ = recover() }()
				handler(msg.Payload)
			}()
		}
	}()
	return func() {
		_ = sub.Close()
		<-done
	}
}

func (c *client) Consume(queueName string, handler func(message string)) func() {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			result, err := c.rc.WithContext(ctx).BLPop(time.Second, queueName).Result()
			if err != nil {
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					return
				}
				if err == redis.Nil {
					continue
				}
				time.Sleep(500 * time.Millisecond)
				continue
			}
			if len(result) > 1 {
				func() {
					defer func() { _ = recover() }()
					handler(result[1])
				}()
			}
		}
	}()
	return func() {
		cancel()
		<-done
	}
}

var NotFound = errors.New("key not found")

func (c *client) Get(key string) (string, error) {
	val, err := c.rc.Get(key).Result()
	if err != nil {
		if err == redis.Nil {
			return "", NotFound
		}
		return "", err
	}
	return val, nil
}

func (c *client) Set(key string, value string, expiry time.Duration) error {
	return c.rc.Set(key, value, expiry).Err()
}

func (c *client) Exists(key string) (bool, error) {
	exists, err := c.rc.Exists(key).Result()
	if err != nil {
		return false, err
	}
	return exists > 0, nil
}

func (c *client) Delete(key string) (bool, error) {
	deleted, err := c.rc.Del(key).Result()
	if err != nil {
		return false, err
	}
	return deleted > 0, nil
}

func (c *client) QueueLength(queueName string) (int, error) {
	length, err := c.rc.LLen(queueName).Result()
	return int(length), err
}
