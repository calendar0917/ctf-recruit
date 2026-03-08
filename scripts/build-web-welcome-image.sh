#!/usr/bin/env bash
set -euo pipefail

docker build -t ctf/web-welcome:dev challenges/templates/web-welcome
