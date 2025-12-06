package producer

import (
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/agbruneau/PubSub/pkg/models"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// TestProduceOrderWithMock vérifie que ProduceOrder appelle correctement le producteur Kafka.
func TestProduceOrderWithMock(t *testing.T) {
	cfg := NewConfig()
	producer := New(cfg)
	mockProducer := new(MockKafkaProducer)
	producer.producer = mockProducer

	// Configurer le mock
	mockProducer.On("Produce", mock.MatchedBy(func(msg *kafka.Message) bool {
		// Vérifier le sujet
		if *msg.TopicPartition.Topic != cfg.Topic {
			return false
		}
		// Vérifier que le message est un JSON valide
		var order models.Order
		if err := json.Unmarshal(msg.Value, &order); err != nil {
			return false
		}
		return true
	}), mock.Anything).Return(nil)

	// Exécuter la méthode
	err := producer.ProduceOrder()

	// Vérifications
	assert.NoError(t, err)
	mockProducer.AssertExpectations(t)
	assert.Equal(t, 2, producer.sequence, "La séquence devrait être incrémentée")
}

// TestProduceOrderError vérifie la gestion des erreurs lors de la production.
func TestProduceOrderError(t *testing.T) {
	cfg := NewConfig()
	producer := New(cfg)
	mockProducer := new(MockKafkaProducer)
	producer.producer = mockProducer

	expectedErr := assert.AnError

	// Configurer le mock pour retourner une erreur
	mockProducer.On("Produce", mock.Anything, mock.Anything).Return(expectedErr)

	// Exécuter la méthode
	err := producer.ProduceOrder()

	// Vérifications
	assert.Error(t, err)
	assert.Contains(t, err.Error(), expectedErr.Error())
	mockProducer.AssertExpectations(t)
	// La séquence ne devrait pas être incrémentée en cas d'erreur de production selon le code actuel ?
	// Vérifions le code: p.sequence++ est appelé APRÈS p.producer.Produce
	// Donc si Produce retourne une erreur, sequence++ N'EST PAS appelé.
	assert.Equal(t, 1, producer.sequence, "La séquence ne devrait pas être incrémentée en cas d'erreur")
}

// TestRun vérifie que Run appelle ProduceOrder en boucle.
// Note: C'est difficile de tester Run directement car c'est une boucle infinie.
// On peut tester une itération ou utiliser un timeout.
func TestRun(t *testing.T) {
	cfg := NewConfig()
	cfg.MessageInterval = 1 * time.Millisecond // Intervalle court pour le test
	producer := New(cfg)
	mockProducer := new(MockKafkaProducer)
	producer.producer = mockProducer

	// On s'attend à ce que Produce soit appelé au moins une fois
	mockProducer.On("Produce", mock.Anything, mock.Anything).Return(nil)

	stopChan := make(chan os.Signal, 1)

	// Démarrer Run dans une goroutine
	go producer.Run(stopChan)

	// Laisser tourner un peu
	time.Sleep(10 * time.Millisecond)

	// Arrêter
	stopChan <- os.Interrupt

	// Attendre que la goroutine finisse (pas de mécanisme de waitgroup dans le code actuel, on assume que ça s'arrête vite)
	time.Sleep(10 * time.Millisecond)

	assert.False(t, producer.running)
}

func TestHandleDeliveryReports(t *testing.T) {
	// Capture stdout to verify logs
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cfg := NewConfig()
	producer := New(cfg)

	// Create channels
	producer.deliveryChan = make(chan kafka.Event, 10)

	// Start handler in background
	go producer.handleDeliveryReports()

	// Send a success message
	topic := "test-topic"
	msgSuccess := &kafka.Message{
		TopicPartition: kafka.TopicPartition{
			Topic:     &topic,
			Partition: 0,
			Offset:    1,
			Error:     nil,
		},
	}
	producer.deliveryChan <- msgSuccess

	// Send a failure message
	msgFail := &kafka.Message{
		TopicPartition: kafka.TopicPartition{
			Topic:     &topic,
			Partition: 0,
			Offset:    0,
			Error:     assert.AnError,
		},
	}
	producer.deliveryChan <- msgFail

	// Allow time for processing
	time.Sleep(50 * time.Millisecond)

	close(producer.deliveryChan)

	// Restore stdout and check content
	w.Close()
	os.Stdout = oldStdout
	out, _ := io.ReadAll(r)
	output := string(out)

	if !strings.Contains(output, "Message delivered to topic") {
		t.Error("Expected success message in logs")
	}
	if !strings.Contains(output, "Message delivery failed") {
		t.Error("Expected failure message in logs")
	}
}

func TestClose(t *testing.T) {
	cfg := NewConfig()
	producer := New(cfg)
	mockProducer := new(MockKafkaProducer)
	producer.producer = mockProducer

	// Flush should be called
	mockProducer.On("Flush", mock.Anything).Return(0)

	// Call Close (rawProducer is nil, but we patched Close to handle it)
	producer.Close()

	mockProducer.AssertExpectations(t)
}
