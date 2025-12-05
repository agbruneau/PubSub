package tracker

import (
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/stretchr/testify/mock"
)

// MockKafkaConsumer est un mock pour l'interface KafkaConsumer.
type MockKafkaConsumer struct {
	mock.Mock
}

func (m *MockKafkaConsumer) SubscribeTopics(topics []string, rebalanceCb kafka.RebalanceCb) error {
	args := m.Called(topics, rebalanceCb)
	return args.Error(0)
}

func (m *MockKafkaConsumer) ReadMessage(timeout time.Duration) (*kafka.Message, error) {
	args := m.Called(timeout)
	msg := args.Get(0)
	if msg == nil {
		return nil, args.Error(1)
	}
	return msg.(*kafka.Message), args.Error(1)
}

func (m *MockKafkaConsumer) Close() error {
	args := m.Called()
	return args.Error(0)
}
