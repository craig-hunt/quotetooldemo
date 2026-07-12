#!/usr/bin/env bash
# Runs `go mod tidy` in each service. Produces go.sum files so Docker builds
# can verify module hashes. Run once after cloning or after adding dependencies.

set -euo pipefail

for svc in customers quotes orders invoices reports; do
  echo "==> tidying services/${svc}"
  (cd "services/${svc}" && go mod tidy)
done

echo "==> done"
