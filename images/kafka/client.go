package kafka

import (
	"context"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/IBM/sarama"
)

type Client interface {
	Publish(topic, key, value string) error
	PublishRaw(topic string, key, value []byte, headers ...Header) error
	Subscribe(topic string, fn func(message Message) (mark string)) (close func())
	Close() error
}

type Message struct {
	Timestamp      time.Time
	BlockTimestamp time.Time
	Key            []byte
	Value          []byte
	Topic          string
	Partition      int64
	Offset         int64
	Headers        []Header
}

type Header struct {
	Key   []byte
	Value []byte
}

func newClient(brokers []string, options Options) (Client, error) {
	cfg := sarama.NewConfig()
	cfg.Producer.Return.Successes = true
	cfg.Producer.RequiredAcks = sarama.WaitForAll
	cfg.Producer.Retry.Max = 5
	cfg.Producer.Idempotent = true
	cfg.Net.MaxOpenRequests = 1
	cfg.Consumer.Return.Errors = true
	cfg.Consumer.Offsets.Initial = options.offsetInitial()
	cfg.Version = sarama.V2_8_0_0
	prod, err := sarama.NewSyncProducer(brokers, cfg)
	if err != nil {
		return nil, err
	}
	cg, err := sarama.NewConsumerGroup(brokers, options.groupId(), cfg)
	if err != nil {
		_ = prod.Close()
		return nil, err
	}
	c := &client{
		brokers:       brokers,
		cfg:           cfg,
		producer:      prod,
		consumerGroup: cg,
		subs:          make(map[string]*subscriber),
		kick:          make(chan struct{}, 1),
	}
	c.ctx, c.cancel = context.WithCancel(context.Background())
	// log consumer group errors - so they don't pile up silently...
	go func() {
		for err := range c.consumerGroup.Errors() {
			log.Printf("kafka: group error: %v", err)
		}
	}()
	// start the consume loop...
	c.loopWG.Add(1)
	go c.run()
	return c, nil
}

type client struct {
	brokers       []string
	cfg           *sarama.Config
	producer      sarama.SyncProducer
	consumerGroup sarama.ConsumerGroup
	// subscriptions
	mu     sync.RWMutex
	subs   map[string]*subscriber // topic -> subscriber (fan-out registry)
	topics []string               // cached keys(subs)
	nextFn int64
	// consume loop lifecycle
	ctx           context.Context
	cancel        context.CancelFunc
	loopWG        sync.WaitGroup
	kick          chan struct{} // signal topic-set changes
	sessMu        sync.Mutex
	sessionCancel context.CancelFunc
}

type fnReg struct {
	id int64
	fn func(Message) (mark string)
}

type subscriber struct {
	mu  sync.RWMutex
	fns []fnReg
}

func newSubscriber() *subscriber { return &subscriber{} }

func (s *subscriber) addFn(id int64, fn func(Message) (mark string)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.fns = append(s.fns, fnReg{id: id, fn: fn})
}

func (s *subscriber) removeFn(id int64) (remaining int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.fns) == 0 {
		return 0
	}
	dst := s.fns[:0]
	for _, r := range s.fns {
		if r.id != id {
			dst = append(dst, r)
		}
	}
	s.fns = append([]fnReg(nil), dst...) // copy-on-write
	return len(s.fns)
}

func (c *client) Publish(topic, key, value string) error {
	_, _, err := c.producer.SendMessage(&sarama.ProducerMessage{
		Topic: topic,
		Key:   sarama.StringEncoder(key),
		Value: sarama.StringEncoder(value),
	})
	return err
}

func (c *client) PublishRaw(topic string, key, value []byte, headers ...Header) error {
	hdrs := make([]sarama.RecordHeader, len(headers))
	for i, h := range headers {
		hdrs[i] = sarama.RecordHeader{Key: h.Key, Value: h.Value}
	}
	_, _, err := c.producer.SendMessage(&sarama.ProducerMessage{
		Topic:   topic,
		Key:     sarama.ByteEncoder(key),
		Value:   sarama.ByteEncoder(value),
		Headers: hdrs,
	})
	return err
}

// Subscribe registers a handler for a topic. Multiple handlers per topic are supported.
//
// The returned closer removes this handler - if it was the last for that topic, the consume
// loop is restarted with the updated topic-set
func (c *client) Subscribe(topic string, fn func(message Message) (mark string)) (closer func()) {
	c.mu.Lock()
	defer c.mu.Unlock()
	sub := c.subs[topic]
	newTopic := false
	if sub == nil {
		sub = newSubscriber()
		c.subs[topic] = sub
		c.rebuildTopicsLocked()
		newTopic = true
	}
	id := atomic.AddInt64(&c.nextFn, 1)
	sub.addFn(id, fn)
	if newTopic {
		c.restartConsumeLoop()
	}
	return func() {
		c.mu.Lock()
		defer c.mu.Unlock()
		if sub.removeFn(id) == 0 {
			// last handler removed - drop the topic entirely...
			delete(c.subs, topic)
			c.rebuildTopicsLocked()
			c.restartConsumeLoop()
		}
	}
}

func (c *client) rebuildTopicsLocked() {
	topics := make([]string, 0, len(c.subs))
	for t := range c.subs {
		topics = append(topics, t)
	}
	c.topics = topics
}

func (c *client) restartConsumeLoop() {
	c.sessMu.Lock()
	if c.sessionCancel != nil {
		c.sessionCancel()
	}
	c.sessMu.Unlock()
	select { // coalesce kicks
	case c.kick <- struct{}{}:
	default:
	}
}

func (c *client) run() {
	defer c.loopWG.Done()
	for {
		// check shutdown
		if c.ctx.Err() != nil {
			return
		}
		// snapshot current topic set and handler
		c.mu.RLock()
		topics := append([]string(nil), c.topics...)
		c.mu.RUnlock()
		// if no topics, wait until we get a kick or shutdown...
		if len(topics) == 0 {
			select {
			case <-c.kick:
				continue
			case <-c.ctx.Done():
				return
			}
		}
		handler := &router{c: c}
		// create a per-iteration session context we can cancel to apply topic changes quickly...
		sessionCtx, sessionCancel := context.WithCancel(c.ctx)
		c.sessMu.Lock()
		c.sessionCancel = sessionCancel
		c.sessMu.Unlock()
		err := c.consumerGroup.Consume(sessionCtx, topics, handler)
		// clear/close the session cancel to avoid leaking it...
		c.sessMu.Lock()
		c.sessionCancel = nil
		c.sessMu.Unlock()
		sessionCancel()
		// if shutting down - exit...
		if c.ctx.Err() != nil {
			return
		}
		if err != nil {
			log.Printf("kafka: consume returned error: %v", err)
			time.Sleep(300 * time.Millisecond)
		}
		// drain one kick if present to coalesce restarts...
		select {
		case <-c.kick:
		default:
		}
	}
}

func (c *client) Close() (err error) {
	// stop the loop first...
	c.cancel()
	c.restartConsumeLoop() // ensure any active session exits
	c.loopWG.Wait()
	// close consumer group and producer...
	if e := c.consumerGroup.Close(); e != nil {
		err = e
	}
	if e := c.producer.Close(); e != nil && err == nil {
		err = e
	}
	return err
}

var _ Client = (*client)(nil)

type router struct {
	c *client
}

func (r *router) Setup(sess sarama.ConsumerGroupSession) error {
	return nil
}

func (r *router) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

func (r *router) ConsumeClaim(sess sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	// identify subscriber for this topic once (sub is stable via r.c.subs map).
	topic := claim.Topic()
	sub := r.getSubscriber(topic)
	if sub == nil {
		// no subscriber (race with removal) - just drain until session ends...
		for {
			select {
			case <-sess.Context().Done():
				return nil
			case _, ok := <-claim.Messages():
				if !ok {
					return nil
				}
			}
		}
	}
	for {
		select {
		case <-sess.Context().Done():
			return nil
		case msg, ok := <-claim.Messages():
			if !ok || msg == nil {
				return nil
			}
			m := translateConsumerMsg(msg)
			// snapshot handlers safely...
			sub.mu.RLock()
			snapshot := make([]fnReg, len(sub.fns))
			copy(snapshot, sub.fns)
			sub.mu.RUnlock()
			// first non-empty mark wins...
			var mark string
			for _, r := range snapshot {
				func() {
					defer func() {
						if rec := recover(); rec != nil {
							log.Printf("kafka: handler panic recovered: %v", rec)
						}
					}()
					if mm := r.fn(m); mm != "" && mark == "" {
						mark = mm
					}
				}()
			}
			sess.MarkMessage(msg, mark)
		}
	}
}

func (r *router) getSubscriber(topic string) *subscriber {
	r.c.mu.RLock()
	defer r.c.mu.RUnlock()
	return r.c.subs[topic]
}

func translateConsumerMsg(msg *sarama.ConsumerMessage) Message {
	return Message{
		Timestamp:      msg.Timestamp,
		BlockTimestamp: time.Now(), // "processed at" (Kafka doesn't set BlockTimestamp on ConsumerMessage)
		Key:            append([]byte(nil), msg.Key...),
		Value:          append([]byte(nil), msg.Value...),
		Topic:          msg.Topic,
		Partition:      int64(msg.Partition),
		Offset:         msg.Offset,
		Headers:        translateConsumerMsgHdrs(msg.Headers),
	}
}

func translateConsumerMsgHdrs(hdrs []*sarama.RecordHeader) []Header {
	if len(hdrs) == 0 {
		return nil
	}
	result := make([]Header, len(hdrs))
	for i, h := range hdrs {
		result[i] = Header{
			Key:   append([]byte(nil), h.Key...),
			Value: append([]byte(nil), h.Value...),
		}
	}
	return result
}
