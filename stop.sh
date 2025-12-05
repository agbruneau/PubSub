#!/bin/bash

# ==============================================================================
# SCRIPT D'ARRÊT PROPRE DE L'APPLICATION KAFKA DEMO
# ==============================================================================
#
# Ce script est conçu pour arrêter proprement tous les composants de l'application.
# Il suit une approche en plusieurs étapes pour s'assurer que les données en
# transit sont traitées avant l'arrêt complet.
#
# Étapes exécutées :
# 1. Arrêt des processus Go :
#    a. Envoi d'un signal SIGTERM : Ce signal demande aux processus Go de
#       s'arrêter proprement. Le producteur videra son tampon et le
#       consommateur terminera de traiter le message en cours.
#    b. Période de grâce : Le script attend jusqu'à 15 secondes pour laisser
#       le temps aux applications de se terminer d'elles-mêmes.
#    c. Arrêt forcé (si nécessaire) : Si les processus sont toujours actifs
#       après le délai, un signal SIGKILL est envoyé pour les forcer à
#       s'arrêter. C'est une mesure de sécurité.
# 2. Arrêt des conteneurs Docker : Une fois les applications Go terminées,
#    `docker compose down` est appelé pour arrêter et supprimer les conteneurs
#    Kafka.
#
# ------------------------------------------------------------------------------

# Active le mode "verbose" pour afficher chaque commande.
set -x

# Obtenir le répertoire du script
script_dir=$(dirname "$0")

# Fonction pour arrêter un processus proprement par son PID
# Prend en paramètre le nom du service et son PID
shutdown_process() {
    local service_name=$1
    local pid=$2

    if ! kill -0 $pid 2>/dev/null; then
        echo "   ℹ️  $service_name (PID: $pid) est déjà arrêté."
        return 0
    fi

    echo "   -> Arrêt de $service_name (PID: $pid)..."
    # Envoi du signal SIGTERM pour un arrêt gracieux
    kill -TERM $pid 2>/dev/null || true

    # Période de grâce de 15 secondes
    local waited=0
    while [ $waited -lt 15 ]; do
        if ! kill -0 $pid 2>/dev/null; then
            echo "   ✅ $service_name s'est arrêté proprement."
            return 0
        fi
        sleep 1
        waited=$((waited + 1))
        echo -n "."
    done
    echo ""

    # Si le processus est toujours là, on force l'arrêt
    echo "   ⚠️  $service_name ne s'est pas arrêté à temps. Arrêt forcé (SIGKILL)..."
    kill -KILL $pid 2>/dev/null || true
    return 1
}

# Étape 1: Arrêter proprement les processus Go (producer PUIS tracker)
echo "🔴 Arrêt séquentiel des processus applicatifs Go..."

producer_pid=""
tracker_pid=""

if [ -f "$script_dir/producer.pid" ]; then
    producer_pid=$(cat "$script_dir/producer.pid")
fi

if [ -f "$script_dir/tracker.pid" ]; then
    tracker_pid=$(cat "$script_dir/tracker.pid")
fi

# 1. Arrêter le producer d'abord pour stopper l'envoi de nouveaux messages
if [ -n "$producer_pid" ]; then
    echo "   1. Arrêt du producer..."
    shutdown_process "Producer" $producer_pid
    rm -f "$script_dir/producer.pid"
fi

# 2. Ensuite, arrêter le tracker pour qu'il traite les messages restants
if [ -n "$tracker_pid" ]; then
    echo "   2. Arrêt du tracker..."
    shutdown_process "Tracker" $tracker_pid
    rm -f "$script_dir/tracker.pid"
fi

# Nettoyage de secours (si les PID files étaient absents ou incorrects)
echo "   🧹 Nettoyage de sécurité (pkill)..."
pkill -TERM -f "./bin/producer" 2>/dev/null || true
pkill -TERM -f "./bin/tracker" 2>/dev/null || true
sleep 2
pkill -KILL -f "./bin/producer" 2>/dev/null || true
pkill -KILL -f "./bin/tracker" 2>/dev/null || true


# Étape 2: Arrêter et supprimer les conteneurs Docker
echo "🔴 Arrêt et suppression des conteneurs Docker..."
sudo docker compose down

echo "✅ L'environnement a été complètement arrêté."
