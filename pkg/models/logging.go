/*
Package models définit les structures de données partagées pour le système PubSub.

Ce fichier contient les structures pour la journalisation structurée et la piste d'audit,
utilisées par les composants tracker et log_monitor.
*/
package models

import "encoding/json"

// LogLevel définit les niveaux de sévérité pour les journaux structurés.
type LogLevel string

const (
	// LogLevelINFO représente un niveau de journalisation informatif.
	LogLevelINFO LogLevel = "INFO"
	// LogLevelERROR représente un niveau de journalisation d'erreur.
	LogLevelERROR LogLevel = "ERROR"
)

// LogEntry est la structure d'un journal écrit dans `tracker.log`.
// Elle est conçue pour le modèle "Application Health Monitoring".
// Chaque entrée est un journal structuré (JSON) contenant des informations sur
// l'état de l'application (démarrage, arrêt, erreurs, métriques). Ce format est optimisé
// pour l'ingestion, l'analyse et la visualisation par des outils de surveillance et d'alerte.
type LogEntry struct {
	Timestamp string                 `json:"timestamp"`          // Horodatage du journal au format RFC3339.
	Level     LogLevel               `json:"level"`              // Niveau de sévérité (INFO, ERROR).
	Message   string                 `json:"message"`            // Message principal du journal.
	Service   string                 `json:"service"`            // Nom du service émetteur.
	Error     string                 `json:"error,omitempty"`    // Message d'erreur, le cas échéant.
	Metadata  map[string]interface{} `json:"metadata,omitempty"` // Données contextuelles supplémentaires.
}

// EventEntry est la structure d'un événement écrit dans `tracker.events`.
// Elle implémente le modèle "Audit Trail" en capturant une copie fidèle et immuable
// de chaque message reçu de Kafka, avec ses métadonnées.
//
// Chaque entrée contient le message brut, le résultat de la désérialisation
// et des informations contextuelles comme le sujet, la partition et le décalage (offset).
// Ce journal est la source de vérité pour l'audit, la relecture d'événements et le débogage.
type EventEntry struct {
	Timestamp      string          `json:"timestamp"`            // Horodatage de réception au format RFC3339.
	EventType      string          `json:"event_type"`           // Type d'événement (ex: "message.received").
	KafkaTopic     string          `json:"kafka_topic"`          // Sujet Kafka source.
	KafkaPartition int32           `json:"kafka_partition"`      // Partition Kafka source.
	KafkaOffset    int64           `json:"kafka_offset"`         // Décalage (offset) du message dans la partition.
	RawMessage     string          `json:"raw_message"`          // Contenu brut du message.
	MessageSize    int             `json:"message_size"`         // Taille du message en octets.
	Deserialized   bool            `json:"deserialized"`         // Indique si la désérialisation a réussi.
	Error          string          `json:"error,omitempty"`      // Erreur de désérialisation, le cas échéant.
	OrderFull      json.RawMessage `json:"order_full,omitempty"` // Contenu complet de la commande désérialisée.
}
