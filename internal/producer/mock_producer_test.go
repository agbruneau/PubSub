package producer

import (
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/stretchr/testify/mock"
)

// MockKafkaProducer est un mock pour l'interface KafkaProducer.
type MockKafkaProducer struct {
	mock.Mock
}

func (m *MockKafkaProducer) Produce(msg *kafka.Message, deliveryChan chan kafka.Event) error {
	args := m.Called(msg, deliveryChan)
	// Simuler l'envoi d'un événement de livraison si deliveryChan n'est pas nil
	if deliveryChan != nil {
		go func() {
			deliveryChan <- &kafka.Message{
				TopicPartition: kafka.TopicPartition{
					Topic:     msg.TopicPartition.Topic,
					Partition: 0,
					Offset:    0,
					Error:     nil,
				},
			}
		}()
	}
	return args.Error(0)
}

func (m *MockKafkaProducer) Flush(timeoutMs int) int {
	args := m.Called(timeoutMs)
	return args.Int(0)
}

func (m *MockKafkaProducer) Close() {
	m.Called()
}
