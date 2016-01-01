#!/bin/sh

# Create the polochon user if needed
[[ "${UID:-""}" =~ ^[0-9]+$ ]] && usermod -u $UID polochon
[[ "${GID:-""}" =~ ^[0-9]+$ ]] && groupmod -g $GID polochon

# Run polochon app as the polochon user
exec sudo -u polochon \
    /usr/bin/polochon -configPath $POLOCHON_CONFIG -tokenPath $POLOCHON_TOKEN
