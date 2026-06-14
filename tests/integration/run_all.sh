#!/bin/bash
set -e
echo "=== Cleaning previous state ==="
docker compose down -v 2>/dev/null || true
echo "=== Starting services ==="
docker compose up -d postgres api
echo "=== Waiting for API to be healthy ==="
for i in $(seq 1 30); do
  if curl -s http://localhost:8080/health > /dev/null 2>&1; then
    echo "API is ready"
    break
  fi
  if [ "$i" -eq 30 ]; then
    echo "Error: API did not start in time"
    docker compose logs api
    exit 1
  fi
  sleep 1
done
echo "=== Running Hurl tests ==="
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR"
rm -rf build
mkdir -p build/report
hurl --test --report-html build/report --variable base_url=http://localhost:8080 *.hurl
HURL_EXIT=$?
echo "=== Tests complete ==="
echo "Report: $SCRIPT_DIR/build/report/index.html"
cd "$SCRIPT_DIR/../.."
docker compose down -v
exit $HURL_EXIT
