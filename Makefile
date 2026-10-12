.PHONY: build run test generate migrate migrate-down migrate-status migrate-validate

build:
	go build ./...

define load_env
	tripgo_original_env=$$(export -p); \
	set -a; \
	for tripgo_env_file in $(1); do \
		if [ -f "$$tripgo_env_file" ]; then \
			. "./$$tripgo_env_file"; \
		fi; \
	done; \
	set +a; \
	eval "$$tripgo_original_env"; \
	unset tripgo_original_env tripgo_env_file;
endef

run:
	@set -eu; \
	$(call load_env,.env.example .env) \
	go run ./cmd/trip-service

test:
	go test -race ./...

generate:
	go tool oapi-codegen \
		-generate types,chi-server \
		-include-operation-ids createTrip,getTrip,finishTrip,health,ready \
		-package api \
		-o internal/generated/api.gen.go \
		contracts/openapi/trip-service.openapi.yaml

define goose
	@set -eu; \
	$(call load_env,.env) \
	: "$${DATABASE_URL:?Set DATABASE_URL in the environment or .env}"; \
	GOOSE_DRIVER=postgres GOOSE_DBSTRING="$$DATABASE_URL" \
		go tool goose -env none -dir migrations $(1)
endef

migrate:
	$(call goose,up)

migrate-down:
	$(call goose,down)

migrate-status:
	$(call goose,status)

migrate-validate:
	go tool goose -env none -dir migrations validate
