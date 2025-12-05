#!/bin/bash

# ==============================================================================
# SCRIPT D'ARRÃŠT PROPRE DE L'APPLICATION KAFKA DEMO
# ==============================================================================
#
# Ce script est conÃ§u pour arrÃªter proprement tous les composants de l'application.
# Il suit une approche en plusieurs Ã©tapes pour s'assurer que les donnÃ©es en
# transit sont traitÃ©es avant l'arrÃªt complet.
#
# Ã‰tapes exÃ©cutÃ©es :
# 1. ArrÃªt des processus Go :
#    a. ArrÃªt du moniteur TUI (s'il est en cours d'exÃ©cution)
#    b. ArrÃªt du producteur (envoi signal SIGTERM)
#    c. ArrÃªt du tracker (consommateur) aprÃ¨s le producteur
#    d. PÃ©riode de grÃ¢ce de 15 secondes pour un arrÃªt propre
#    e. ArrÃªt forcÃ© (SIGKILL) si nÃ©cessaire
# 2. Nettoyage de secours avec pkill
# 3. ArrÃªt des conteneurs Docker
#
# Note: Ce script est appelÃ© automatiquement par start.sh lorsque le moniteur
#       se termine (sortie via 'q' ou Ctrl+C).
#
# ------------------------------------------------------------------------------

# Active le mode "verbose" pour afficher chaque commande.
set -x

# Obtenir le rÃ©pertoire du script
script_dir=$(dirname "$0")

# Fonction pour arrÃªter un processus proprement par son PID
# Prend en paramÃ¨tre le nom du service et son PID
shutdown_process() {
    local service_name=$1
    local pid=$2

    if ! kill -0 $pid 2>/dev/null; then
        echo "   â„¹ï¸  $service_name (PID: $pid) est dÃ©jÃ  arrÃªtÃ©."
        return 0
    fi

    echo "   -> ArrÃªt de $service_name (PID: $pid)..."
    # Envoi du signal SIGTERM pour un arrÃªt gracieux
    kill -TERM $pid 2>/dev/null || true

    # PÃ©riode de grÃ¢ce de 15 secondes
    local waited=0
    while [ $waited -lt 15 ]; do
        if ! kill -0 $pid 2>/dev/null; then
            echo "   âœ… $service_name s'est arrÃªtÃ© proprement."
            return 0
        fi
        sleep 1
        waited=$((waited + 1))
        echo -n "."
    done
    echo ""

    # Si le processus est toujours lÃ , on force l'arrÃªt
    echo "   âš ï¸  $service_name ne s'est pas arrÃªtÃ© Ã  temps. ArrÃªt forcÃ© (SIGKILL)..."
    kill -KILL $pid 2>/dev/null || true
    return 1
}

# Ã‰tape 1: ArrÃªter proprement les processus Go (monitor, producer PUIS tracker)
echo "ðŸ”´ ArrÃªt sÃ©quentiel des processus applicatifs Go..."

monitor_pid=""
producer_pid=""
tracker_pid=""

if [ -f "$script_dir/monitor.pid" ]; then
    monitor_pid=$(cat "$script_dir/monitor.pid")
fi

if [ -f "$script_dir/producer.pid" ]; then
    producer_pid=$(cat "$script_dir/producer.pid")
fi

if [ -f "$script_dir/tracker.pid" ]; then
    tracker_pid=$(cat "$script_dir/tracker.pid")
fi

# 1. ArrÃªter le moniteur d'abord (si lancÃ© sÃ©parÃ©ment)
if [ -n "$monitor_pid" ]; then
    echo "   1. ArrÃªt du moniteur..."
    shutdown_process "Monitor" $monitor_pid
    rm -f "$script_dir/monitor.pid"
fi

# 2. ArrÃªter le producer pour stopper l'envoi de nouveaux messages
if [ -n "$producer_pid" ]; then
    echo "   2. ArrÃªt du producer..."
    shutdown_process "Producer" $producer_pid
    rm -f "$script_dir/producer.pid"
fi

# 3. Ensuite, arrÃªter le tracker pour qu'il traite les messages restants
if [ -n "$tracker_pid" ]; then
    echo "   3. ArrÃªt du tracker..."
    shutdown_process "Tracker" $tracker_pid
    rm -f "$script_dir/tracker.pid"
fi

# Nettoyage de secours (si les PID files Ã©taient absents ou incorrects)
echo "   ðŸ§¹ Nettoyage de sÃ©curitÃ© (pkill)..."
pkill -TERM -f "./bin/monitor" 2>/dev/null || true
pkill -TERM -f "./bin/producer" 2>/dev/null || true
pkill -TERM -f "./bin/tracker" 2>/dev/null || true
sleep 2
pkill -KILL -f "./bin/monitor" 2>/dev/null || true
pkill -KILL -f "./bin/producer" 2>/dev/null || true
pkill -KILL -f "./bin/tracker" 2>/dev/null || true


# Ã‰tape 2: ArrÃªter et supprimer les conteneurs Docker
echo "ðŸ”´ ArrÃªt et suppression des conteneurs Docker..."
sudo docker compose down

echo "âœ… L'environnement a Ã©tÃ© complÃ¨tement arrÃªtÃ©."
