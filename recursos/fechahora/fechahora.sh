#!/bin/bash
# Agrega la fecha y hora actuales (y el nombre del equipo) a un archivo cada 5 segundos.
# Uso: fechahora.sh [archivo]     Por defecto: /var/log/fechahora.log
# Para verlo funcionar en la VM:  tail -f /var/log/fechahora.log

ARCHIVO="${1:-/var/log/fechahora.log}"

while true; do
    echo "$(date '+%Y-%m-%d %H:%M:%S') - $(hostname)" >> "$ARCHIVO"
    sleep 5
done
