# Système de Suivi de Commandes Kafka (Kafka Order Tracking System)

Bienvenue dans le projet de démonstration **Kafka Order Tracking**. Ce projet est une implémentation de référence en **Go** illustrant une architecture événementielle (EDA) robuste utilisant **Apache Kafka**. Il simule un flux de commandes e-commerce complet, de la production à la consommation, avec une observabilité avancée.

## 📋 Table des Matières

- [Fonctionnalités et Patterns](#-fonctionnalités-et-patterns)
- [Prérequis](#-prérequis)
- [Démarrage Rapide](#-démarrage-rapide)
- [Utilisation et Monitoring](#-utilisation-et-monitoring)
- [Arrêt du Système](#-arrêt-du-système)
- [Configuration](#-configuration)
- [Structure du Projet](#-structure-du-projet)
- [Développement et Tests](#-développement-et-tests)

---

## 🌟 Fonctionnalités et Patterns

Ce projet met en œuvre les meilleures pratiques de l'ingénierie logicielle distribuée :

- **Event-Driven Architecture (EDA)** : Découplage total entre le producteur et le consommateur.
- **Event Carried State Transfer (ECST)** : Les messages contiennent tout le contexte nécessaire.
- **Retry Pattern** : Backoff exponentiel avec jitter pour gérer les erreurs transitoires.
- **Dead Letter Queue (DLQ)** : Messages en échec envoyés vers `orders-dlq` pour analyse.
- **Graceful Shutdown** : Gestion propre des signaux (SIGTERM, SIGINT).
- **Configuration Externe** : Fichier YAML + variables d'environnement.

---

## 🛠 Prérequis

Avant de commencer, assurez-vous d'avoir installé :

1.  **Docker** et **Docker Compose** (V2).
2.  **Go** (version 1.22 ou supérieure).
3.  **GCC/MinGW** (ou Docker pour la compilation) - requis pour `confluent-kafka-go`.
4.  Un terminal compatible ANSI (pour le moniteur).
5.  Privilèges `sudo` (requis pour les commandes Docker dans les scripts).

---

## 🚀 Démarrage Rapide

### Méthode 1 : Script Automatisé (Recommandé)

Le projet fournit des scripts pour gérer le cycle de vie complet :

```bash
# 1. Démarrer l'environnement complet
./start.sh
```

**Ce que fait `start.sh` :**

1. ✅ Démarre le conteneur Kafka via Docker Compose
2. ✅ Attend que Kafka soit prêt (attente active)
3. ✅ Crée le topic `orders` automatiquement
4. ✅ Compile les binaires dans `bin/`
5. ✅ Lance le **Tracker** (consommateur) en arrière-plan
6. ✅ Lance le **Producer** (producteur) en arrière-plan

```bash
# 2. Arrêter proprement l'environnement
./stop.sh
```

**Ce que fait `stop.sh` :**

1. ✅ Envoie SIGTERM au producer (arrêt gracieux)
2. ✅ Envoie SIGTERM au tracker (traite les messages restants)
3. ✅ Attend la fin des processus (timeout 15s)
4. ✅ Arrêt forcé si nécessaire (SIGKILL)
5. ✅ Arrête les conteneurs Docker

### Méthode 2 : Lancement Manuel

Si vous préférez un contrôle plus fin :

```bash
# Terminal 1 : Démarrer Kafka
docker compose up -d

# Attendre que Kafka soit prêt, puis créer le topic
docker exec kafka kafka-topics --bootstrap-server localhost:9092 \
  --create --if-not-exists --topic orders --partitions 1 --replication-factor 1

# Terminal 2 : Compiler et lancer le tracker
go build -tags kafka -o bin/tracker ./cmd/tracker
./bin/tracker

# Terminal 3 : Compiler et lancer le producer
go build -tags kafka -o bin/producer ./cmd/producer
./bin/producer

# Terminal 4 : Lancer le moniteur
go build -o bin/monitor ./cmd/monitor
./bin/monitor
```

---

## 📊 Utilisation et Monitoring

Une fois le système lancé, plusieurs méthodes s'offrent à vous pour observer l'activité.

### 1. Le Moniteur Interactif (Recommandé)

Lancez le moniteur dans un **nouveau terminal** :

```bash
./bin/monitor
```

- **Touches** : `q` ou `Ctrl+C` pour quitter.
- **Fonctionnalités** : Affiche le débit (msg/sec), le taux de succès, et les derniers logs.

### 2. Observation des Logs Bruts

```bash
# Activité métier (Audit)
tail -f tracker.events

# Santé technique (Logs JSON)
tail -f tracker.log | jq
```

---

## 🛑 Arrêt du Système

```bash
./stop.sh
```

Ce script :

- Utilise les fichiers PID (`producer.pid`, `tracker.pid`) pour identifier les processus
- Envoie SIGTERM pour un arrêt gracieux
- Laisse le temps aux applications de terminer (flush des messages)
- Arrête l'infrastructure Docker

---

## ⚙️ Configuration

### Fichier de Configuration

Copiez le template et personnalisez :

```bash
cp config.yaml.example config.yaml
```

Options principales :

```yaml
kafka:
  broker: "localhost:9092" # Adresse du broker
  topic: "orders" # Topic Kafka
  consumer_group: "order-tracker-group"

producer:
  interval_ms: 2000 # Intervalle entre messages

retry:
  max_attempts: 3 # Tentatives max avant DLQ
  initial_delay_ms: 100 # Délai initial
  multiplier: 2.0 # Multiplicateur backoff

dlq:
  enabled: true # Activer Dead Letter Queue
  topic: "orders-dlq"
```

### Variables d'Environnement

Les variables d'environnement surchargent le fichier YAML :

| Variable               | Description               |
| ---------------------- | ------------------------- |
| `KAFKA_BROKER`         | Adresse du broker Kafka   |
| `KAFKA_TOPIC`          | Nom du topic              |
| `PRODUCER_INTERVAL_MS` | Intervalle entre messages |
| `RETRY_MAX_ATTEMPTS`   | Nombre max de tentatives  |
| `DLQ_ENABLED`          | Activer/désactiver DLQ    |

---

## 📂 Structure du Projet

```
PubSub/
├── cmd/                           # Points d'entrée
│   ├── producer/main.go
│   ├── tracker/main.go
│   └── monitor/main.go
├── internal/                      # Paquets privés
│   ├── config/                   # Configuration
│   │   ├── config.go            # Constantes
│   │   └── loader.go            # Chargeur YAML/env
│   ├── producer/                 # Logique producteur
│   ├── tracker/                  # Logique consommateur
│   ├── monitor/                  # Logique TUI
│   └── retry/                    # Retry + DLQ
│       ├── retry.go             # Backoff exponentiel
│       └── dlq.go               # Dead Letter Queue
├── pkg/models/                    # Modèles partagés
│   ├── order.go
│   └── logging.go
├── bin/                           # Binaires (généré)
├── start.sh                       # Démarrage automatisé
├── stop.sh                        # Arrêt gracieux
├── config.yaml.example            # Template configuration
└── docker-compose.yaml            # Kafka Docker
```

---

## 💻 Développement et Tests

### Compilation

```bash
# Tous les binaires
make build

# Individuellement
make build-producer   # bin/producer
make build-tracker    # bin/tracker
make build-monitor    # bin/monitor
```

### Tests

```bash
# Tous les tests
make test

# Tests par package
go test -v ./pkg/models/...      # Modèles
go test -v ./internal/retry/...  # Retry pattern
go test -tags kafka -v ./internal/producer/...  # Producer
```

> **Note CGO** : Les packages `producer` et `tracker` utilisent `confluent-kafka-go` qui nécessite CGO. Pour compiler sur Windows sans GCC, utilisez Docker :
>
> ```bash
> docker run --rm -v $(pwd):/app -w /app golang:1.22 make build
> ```
