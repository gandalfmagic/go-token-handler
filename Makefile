.PHONY: test
test:
	go test ./... -coverprofile cover.out -json > test-report.json

.PHONY: test-integration-start
test-integration-start:
	docker run --name postgresql-integration-test-api-gateway --rm -d -p 5532:5432 -e POSTGRES_DB=sessions -e POSTGRES_PASSWORD=postgres postgres:15.2-alpine3.17
	while [[ $$(docker logs --tail 1 postgresql-integration-test-api-gateway 2>&1 | grep "database system is ready to accept connections") == "" ]]; do sleep 1; done

.PHONY: test-integration-stop
test-integration-stop:
	docker stop postgresql-integration-test-api-gateway

.PHONY: test-integration
test-integration: test-integration-start
	go test ./... -tags=integration -coverprofile cover.out -json ./... > test-report.json || (docker stop postgresql-integration-test-api-gateway; exit 1)
	docker stop postgresql-integration-test-api-gateway

.PHONY: start-keycloak
start-keycloak:
	docker run --name keycloak-api-gateway --rm -d -p 8080:8080 -v "${PWD}/test/realms":/opt/keycloak/data/import -e KEYCLOAK_ADMIN=admin -e KEYCLOAK_ADMIN_PASSWORD=admin -e PROXY_ADDRESS_FORWARDING=true quay.io/keycloak/keycloak:20.0.3 start-dev --import-realm

.PHONY: stop-keycloak
stop-keycloak:
	docker stop keycloak-api-gateway

.PHONY: log-keycloak
log-keycloak:
	docker logs -f keycloak-api-gateway

.PHONY: start-frontend
start-frontend:
	docker run --name frontend-api-gateway --rm -d -p 8081:80 -v "${PWD}/test/frontend":/usr/share/nginx/html nginx:1.23.3

.PHONY: stop-frontend
stop-frontend:
	docker stop frontend-api-gateway

.PHONY: start-postgresql
start-postgresql:
	docker run --name postgresql-api-gateway --rm -d -p 5432:5432 -e POSTGRES_DB=sessions -e POSTGRES_PASSWORD=postgres postgres:15.2-alpine3.17

.PHONY: stop-postgresql
stop-postgresql:
	docker stop postgresql-api-gateway

.PHONY: log-postgresql
log-postgresql:
	docker logs -f postgresql-api-gateway

.PHONY: start-jaeger
start-jaeger:
	docker run --name jaeger-api-gateway --rm -d -p 5775:5775/udp -p 6831:6831/udp -p 6832:6832/udp -p 5778:5778 -p 16686:16686 -p 14268:14268 -p 9411:9411 -e COLLECTOR_ZIPKIN_HTTP_PORT=9411 jaegertracing/all-in-one:1.6

.PHONY: stop-jaeger
stop-jaeger:
	docker stop jaeger-api-gateway

.PHONY: log-jaeger
log-jaeger:
	docker logs -f jaeger-api-gateway

.PHONY: start-all
start-all: start-frontend start-jaeger start-keycloak start-postgresql

.PHONY: stop-all
stop-all: stop-frontend stop-jaeger stop-keycloak stop-postgresql
