#!/bin/sh

# Create the polochon user if needed
[ "$POLOCHON_UID" ] && usermod -u "$POLOCHON_UID" polochon
[ "$POLOCHON_GID" ] && groupmod -g "$POLOCHON_GID" polochon

# Run polochon app as the polochon user
exec su -c "/usr/bin/polochon -configPath $POLOCHON_CONFIG -tokenPath $POLOCHON_TOKEN" - polochon
