.PHONY: build run test generate migrate migrate-down migrate-status migrate-validate

build:
	go build ./...

run:
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
	unset DATABASE_URL; \
	. ./.env; \
	: "$${DATABASE_URL:?Set DATABASE_URL in .env}"; \
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
