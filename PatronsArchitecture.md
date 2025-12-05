# 🏗️ Architecture du Projet Kafka Demo

Ce document détaille les modèles d'architecture et de conception implémentés dans ce projet. Il sert de référence pour comprendre les décisions techniques et la structure du code.

## 🎯 Vue d'ensemble

Le projet est une démonstration d'une architecture orientée événements (EDA) utilisant Apache Kafka. Il simule un système de traitement de commandes e-commerce avec une séparation stricte entre la production de données, leur consommation et leur surveillance.

## 🧩 Patrons d'Architecture Implémentés

### 1. Event-Driven Architecture (EDA)
Le système repose entièrement sur l'échange de messages asynchrones.
- **Implémentation** : Kafka agit comme l'épine dorsale. Le producteur et le consommateur sont découplés et ne communiquent que via le topic `orders`.
- **Fichiers** : `producer.go` (émetteur), `tracker.go` (récepteur).

### 2. Publisher/Subscriber (Pub/Sub)
Le modèle de communication est de type "un-vers-plusieurs" (bien qu'ici nous ayons un seul consommateur principal pour la démo, l'architecture permet d'en ajouter d'autres sans modifier le producteur).
- **Implémentation** : Le `producer` publie sur le topic `orders`. Le `tracker` s'abonne à ce même topic via un `consumer_group`.
- **Fichiers** : `producer.go` (Publish), `tracker.go` (Subscribe).

### 3. Event Carried State Transfer (ECST)
Les événements transportent l'intégralité de l'état nécessaire au traitement, et pas seulement une notification de changement (comme un simple ID).
- **Bénéfice** : Le consommateur n'a pas besoin de rappeler le producteur (ou une base de données) pour enrichir les données, ce qui améliore la performance et le découplage.
- **Implémentation** : La structure `Order` contient toutes les infos (Client, Items, Paiement, Adresses).
- **Preuve** : `producer.go` (méthode `GenerateOrder` crée un objet complet).

### 4. Guaranteed Delivery (At-Least-Once)
Le producteur s'assure que le message a bien été reçu par le broker.
- **Implémentation** : Utilisation d'un canal de rapport de livraison (`deliveryChan`). Le producteur attend l'accusé de réception (ACK) du broker.
- **Fichiers** : `producer.go` (fonction `handleDeliveryReports` et usage de `Flush`).

### 5. Graceful Shutdown
Les applications interceptent les signaux du système d'exploitation (SIGINT, SIGTERM) pour s'arrêter proprement.
- **Bénéfice** : Évite la perte de données en mémoire et la corruption de fichiers.
- **Implémentation** :
    - **Producer** : Appelle `Flush()` pour envoyer les messages restants dans le buffer.
    - **Tracker** : Termine le traitement du message en cours et ferme les descripteurs de fichiers.
- **Fichiers** : `producer.go` (`Run` avec `stopChan`), `tracker.go` (`Stop`).

### 6. Observabilité & Structured Logging
Le système sépare clairement les logs techniques des logs métier (Audit).
- **Implémentation** :
    - `tracker.log` : Logs JSON pour la santé du service (erreurs, infos de démarrage).
    - `tracker.events` : Journal d'audit append-only contenant les messages bruts reçus.
- **Pattern** : Séparation des préoccupations (Technical Logging vs Audit Logging).
- **Fichiers** : `tracker.go` (types `Logger`, `LogEntry`, `EventEntry`).

### 7. Command Query Responsibility Segregation (CQRS) - *Approche*
Bien que ce ne soit pas un CQRS strict (avec des modèles de données distincts en écriture/lecture), l'architecture sépare le **traitement** (Tracker) de la **visualisation** (Monitor).
- **Implémentation** : `log_monitor.go` lit les fichiers produits par `tracker.go` sans interférer avec le processus de consommation Kafka. Le moniteur est en lecture seule.

### 8. Idempotency (Consumer Side)
Bien que le code de traitement actuel (`displayOrder`) soit idempotent par nature (affichage simple), l'architecture prépare le terrain pour une idempotence réelle via l'utilisation de groupes de consommateurs et la gestion des offsets Kafka.

## 🛠️ Structure du Code

- **Build Tags** : Utilisation de `//go:build` pour générer plusieurs binaires (`producer`, `tracker`, `monitor`) à partir d'une base de code partagée.
- **Shared Models** : Les structures de données (`Order`, `LogEntry`) sont partagées pour garantir la cohérence des contrats de données.

## 🚀 Infrastructure

- **Containerization** : Utilisation de Docker Compose pour orchestrer Kafka (en mode KRaft, sans Zookeeper).
- **Infrastructure as Code (Light)** : La configuration Kafka est définie déclarativement dans `docker-compose.yaml`.
