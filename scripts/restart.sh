#!/usr/bin/env bash

set -euo pipefail
DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &>/dev/null && pwd)"
WORKDIR="$(dirname "${DIR}")"

cd "${WORKDIR}"

overmind restart workerserver
overmind restart hosting
