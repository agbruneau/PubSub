/*
Package config fournit la configuration centralisée pour le système PubSub.

Ce paquet contient toutes les constantes et structures de configuration
partagées entre les composants producer, tracker et monitor.
*/
package config

import "time"

// Configuration par défaut de Kafka
const (
	// DefaultKafkaBroker est l'adresse par défaut du broker Kafka.
	DefaultKafkaBroker = "localhost:9092"
	// DefaultConsumerGroup est le groupe de consommateurs par défaut.
	DefaultConsumerGroup = "order-tracker-group"
	// DefaultTopic est le topic Kafka par défaut.
	DefaultTopic = "orders"
)

// Fichiers de logs
const (
	// TrackerLogFile est le nom du fichier de journalisation structurée.
	TrackerLogFile = "tracker.log"
	// TrackerEventsFile est le nom du fichier d'audit des événements.
	TrackerEventsFile = "tracker.events"
)

// Délais et intervalles communs
const (
	// FlushTimeoutMs est le délai d'expiration par défaut pour le flush des messages (en ms).
	FlushTimeoutMs = 15000
)

// Constantes pour le producteur
const (
	// ProducerMessageInterval définit l'intervalle entre l'envoi de deux messages.
	ProducerMessageInterval = 2 * time.Second
	// ProducerFlushTimeout définit le temps maximum d'attente pour l'envoi des messages à l'arrêt.
	ProducerFlushTimeout = 5 * time.Second
	// ProducerDeliveryChannelSize est la taille du tampon pour les rapports de livraison.
	ProducerDeliveryChannelSize = 10000
	// ProducerDefaultTaxRate est le taux de taxe par défaut.
	ProducerDefaultTaxRate = 0.20
	// ProducerDefaultShippingFee est le frais de port par défaut.
	ProducerDefaultShippingFee = 2.50
	// ProducerDefaultCurrency est la devise par défaut.
	ProducerDefaultCurrency = "EUR"
	// ProducerDefaultPayment est le moyen de paiement par défaut.
	ProducerDefaultPayment = "credit_card"
	// ProducerDefaultWarehouse est l'entrepôt par défaut.
	ProducerDefaultWarehouse = "PARIS-01"
)

// Constantes pour le tracker (consommateur)
const (
	// TrackerMetricsInterval est l'intervalle de calcul des métriques.
	TrackerMetricsInterval = 30 * time.Second
	// TrackerConsumerReadTimeout est le temps d'attente pour la lecture d'un message Kafka.
	TrackerConsumerReadTimeout = 1 * time.Second
	// TrackerMaxConsecutiveErrors est le nombre maximum d'erreurs consécutives tolérées avant alerte.
	TrackerMaxConsecutiveErrors = 3
	// TrackerServiceName est le nom du service pour les logs.
	TrackerServiceName = "order-tracker"
)

// Constantes pour le moniteur de logs
const (
	// MonitorMaxRecentLogs est le nombre maximum de logs récents à conserver en mémoire.
	MonitorMaxRecentLogs = 20
	// MonitorMaxRecentEvents est le nombre maximum d'événements récents à conserver en mémoire.
	MonitorMaxRecentEvents = 20
	// MonitorMaxHistorySize est la taille de l'historique pour les graphiques.
	MonitorMaxHistorySize = 50
	// MonitorLogChannelBuffer est la taille du buffer pour le canal de logs.
	MonitorLogChannelBuffer = 100
	// MonitorEventChannelBuffer est la taille du buffer pour le canal d'événements.
	MonitorEventChannelBuffer = 100

	// Seuils de taux de succès (%)

	// MonitorSuccessRateExcellent définit le seuil pour un taux de succès excellent.
	MonitorSuccessRateExcellent = 95.0
	// MonitorSuccessRateGood définit le seuil pour un taux de succès bon.
	MonitorSuccessRateGood = 80.0

	// Seuils de débit (messages par seconde)

	// MonitorThroughputNormal définit le seuil pour un débit normal.
	MonitorThroughputNormal = 0.3
	// MonitorThroughputLow définit le seuil pour un débit faible.
	MonitorThroughputLow = 0.1

	// Seuils de temps d'erreur

	// MonitorErrorTimeoutCritical est le délai avant qu'une erreur soit considérée critique.
	MonitorErrorTimeoutCritical = 1 * time.Minute
	// MonitorErrorTimeoutWarning est le délai avant qu'une erreur soit considérée comme un avertissement.
	MonitorErrorTimeoutWarning = 5 * time.Minute

	// Seuils de score de qualité pour le débit

	// MonitorQualityThroughputHigh est le seuil pour un score de débit élevé.
	MonitorQualityThroughputHigh = 0.5
	// MonitorQualityThroughputMedium est le seuil pour un score de débit moyen.
	MonitorQualityThroughputMedium = 0.3
	// MonitorQualityThroughputLow est le seuil pour un score de débit faible.
	MonitorQualityThroughputLow = 0.1

	// Seuils de score de qualité global

	// MonitorQualityScoreExcellent est le seuil pour un score de qualité global excellent.
	MonitorQualityScoreExcellent = 90.0
	// MonitorQualityScoreGood est le seuil pour un score de qualité global bon.
	MonitorQualityScoreGood = 70.0
	// MonitorQualityScoreMedium est le seuil pour un score de qualité global moyen.
	MonitorQualityScoreMedium = 50.0

	// Intervalles de temps

	// MonitorFileCheckInterval est l'intervalle de vérification de l'existence des fichiers.
	MonitorFileCheckInterval = 1 * time.Second
	// MonitorFilePollInterval est l'intervalle de lecture des nouveaux contenus dans les fichiers.
	MonitorFilePollInterval = 200 * time.Millisecond
	// MonitorUIUpdateInterval est l'intervalle de rafraîchissement de l'interface utilisateur.
	MonitorUIUpdateInterval = 500 * time.Millisecond

	// Limites d'affichage

	// MonitorMaxLogRowLength est la longueur maximale d'une ligne de log affichée.
	MonitorMaxLogRowLength = 75
	// MonitorMaxEventRowLength est la longueur maximale d'une ligne d'événement affichée.
	MonitorMaxEventRowLength = 75
	// MonitorTruncateSuffix est le suffixe ajouté lorsque le texte est tronqué.
	MonitorTruncateSuffix = "..."
)
