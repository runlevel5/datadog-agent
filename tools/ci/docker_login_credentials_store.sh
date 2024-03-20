#!/usr/bin/env bash
set -euo pipefail
DOCKER_CONFIG="$HOME"/.docker/config.json
if [[ -f "$DOCKER_CONFIG" ]]; then
  sed -i '2i    "credsStore": "ci",' "$DOCKER_CONFIG"
else
  echo "Creating docker config"
  echo '{\n  "credsStore": "ci"\n}' > "$DOCKER_CONFIG"
fi