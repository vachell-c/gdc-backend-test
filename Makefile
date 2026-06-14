.PHONY: build run test test-unit test-integration load-test db-migrate lint clean

build:
	go build -o bin/server ./cmd/server

run:
	@echo "=== Building Docker image ==="
	@docker compose build api 2>/dev/null
	@echo "=== Starting services (postgres + api + otel-lgtm) ==="
	@docker compose up -d postgres api otel-lgtm 2>/dev/null
	@echo "=== Waiting for API to be healthy ==="
	@for i in $$(seq 1 20); do \
		if curl -s http://localhost:8080/health 2>/dev/null | grep -q "ok"; then \
			echo "API ready at http://localhost:8080"; break; \
		fi; \
		if [ "$$i" -eq 20 ]; then echo "API not ready in time"; docker compose logs api; exit 1; fi; \
		sleep 2; \
	done
	@echo "=== Grafana (otel-lgtm) at http://localhost:3000 ==="

db-migrate:
	docker compose exec -T postgres psql -U postgres -d taskdb -f /docker-entrypoint-initdb.d/001_init.sql

test-unit:
	go test ./tests/unit/... -v -count=1 -race

test-integration:
	bash tests/integration/run_all.sh

test: test-unit

load-test:
	@echo "=== Running k6 load test ==="
	k6 run tests/load/load_test.js
	@echo "=== Load test complete ==="
	@echo "Open Grafana at http://localhost:3000 to view traces and metrics"

lint:
	go vet ./...

clean:
	@echo "=== Stopping local server ==="
	@lsof -ti:8080 2>/dev/null | xargs -r kill -9 2>/dev/null || true
	rm -rf bin/
	docker compose down -v
