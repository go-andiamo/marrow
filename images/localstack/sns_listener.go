package localstack

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

func newSnsListener(mappedPort string, options SNSOptions, client *sns.Client, arns map[string]string) (result *snsListener, err error) {
	result = &snsListener{
		actualHost:   "http://host.docker.internal",
		mappedPort:   mappedPort,
		max:          options.MaxMessages,
		jsonMessages: options.JsonMessages,
		unmarshaler:  options.Unmarshaler,
		counts:       make(map[string]int64),
		msgs:         make(map[string][]any),
	}
	if result.unmarshaler == nil {
		result.unmarshaler = result.unmarshalMessage
	}
	if err = result.start(); err != nil {
		return nil, err
	} else {
		defer func() {
			if err != nil {
				result.shutdown()
			}
		}()
		// need to subscribe to topics and wait for subscription confirmations...
		for topic, arn := range arns {
			si := &sns.SubscribeInput{
				Protocol: aws.String("http"),
				TopicArn: aws.String(arn),
				Endpoint: aws.String(result.actualHost + "/" + topic),
			}
			if _, err = client.Subscribe(context.Background(), si); err != nil {
				return nil, err
			} else {
				result.subsWg.Add(1)
			}
		}
	}
	err = result.waitWithTimeout()
	return result, err
}

func (s *snsListener) waitWithTimeout() error {
	done := make(chan struct{})
	go func() {
		s.subsWg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-time.After(10 * time.Second):
		return errors.New("subscription confirmations timeout")
	}
}

type snsListener struct {
	actualHost string
	mappedPort string
	server     *http.Server
	listener   net.Listener
	mutex      sync.RWMutex

	subsWg       sync.WaitGroup
	max          int
	jsonMessages bool
	unmarshaler  func(msg SnsMessage) any
	counts       map[string]int64
	msgs         map[string][]any
}

func (s *snsListener) start() (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("sns listener failed to start: %w", err)
		}
	}()
	// listen on "127.0.0.1:0" (i.e. port 0) tells the OS to pick an unused port
	if s.listener, err = net.Listen("tcp", "127.0.0.1:0"); err == nil {
		addr := s.listener.Addr().(*net.TCPAddr)
		s.actualHost = s.actualHost + ":" + strconv.Itoa(addr.Port)
		s.server = &http.Server{Handler: s}
		go func() {
			_ = s.server.Serve(s.listener)
		}()
	}
	return
}

func (s *snsListener) shutdown() {
	if s.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.server.Shutdown(ctx)
	}
}

type SnsMessage struct {
	Type         string `json:"Type"`
	MessageId    string `json:"MessageId"`
	TopicArn     string `json:"TopicArn"`
	Message      string `json:"Message"`
	Timestamp    string `json:"Timestamp"`
	Token        string `json:"Token"`
	SubscribeURL string `json:"SubscribeURL"`
}

func (s *snsListener) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer func() {
		_ = r.Body.Close()
	}()
	status := http.StatusBadRequest
	var msg SnsMessage
	if err := json.NewDecoder(r.Body).Decode(&msg); err == nil {
		status = http.StatusOK
		switch msg.Type {
		case "SubscriptionConfirmation":
			s.confirmSubscription(msg)
		case "Notification":
			s.handleMessage(strings.TrimPrefix(r.URL.Path, "/"), msg)
		}
	}
	w.WriteHeader(status)
}

func (s *snsListener) handleMessage(topic string, msg SnsMessage) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.counts[topic]++
	if s.max > 0 {
		actualMsg := s.unmarshaler(msg)
		if msgs, ok := s.msgs[topic]; ok {
			if len(msgs) < s.max {
				msgs = append(msgs, actualMsg)
			} else {
				// drop oldest and append newest (nil for clarity - copy + overwrite already releases)
				msgs[0] = nil
				copy(msgs, msgs[1:])
				msgs[len(msgs)-1] = actualMsg
			}
			s.msgs[topic] = msgs
		} else {
			s.msgs[topic] = []any{actualMsg}
		}
	}
}

func (s *snsListener) unmarshalMessage(msg SnsMessage) any {
	result := map[string]any{
		"Type":      msg.Type,
		"MessageId": msg.MessageId,
		"TopicArn":  msg.TopicArn,
		"Message":   msg.Message,
		"Timestamp": msg.Timestamp,
		"Token":     msg.Token,
	}
	if s.jsonMessages {
		var jmsg any
		if err := json.Unmarshal([]byte(msg.Message), &jmsg); err == nil {
			result["Message"] = jmsg
		}
	}
	return result
}

func (s *snsListener) confirmSubscription(msg SnsMessage) {
	url := strings.Replace(msg.SubscribeURL, "localhost:4566", "localhost:"+s.mappedPort, 1)
	client := &http.Client{
		Timeout: time.Second * 5,
	}
	res, err := client.Get(url)
	if err != nil {
		fmt.Println("Error confirming SNS subscription: ", err)
	} else {
		defer func() {
			_ = res.Body.Close()
		}()
		if res.StatusCode != http.StatusOK {
			fmt.Println("Error confirming SNS subscription - status code: ", res.StatusCode)
		} else {
			s.subsWg.Done()
		}
	}
}

func (s *snsListener) messagesCount(topic string) (count int64) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if topic == "" || topic == "*" {
		for _, c := range s.counts {
			count += c
		}
	} else if c, ok := s.counts[topic]; ok {
		count = c
	}
	return count
}

func (s *snsListener) messages(topic string) (result []any) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if msgs, ok := s.msgs[topic]; ok {
		result = make([]any, len(msgs))
		copy(result, msgs)
	}
	return result
}
