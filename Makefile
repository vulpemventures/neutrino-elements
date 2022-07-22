.PHONY: help fmt test clean build build pg droppg createdb dropdb createtestdb \
droptestdb recreatedb recreatetestdb psql wpkh

## help: prints this help message
help:
	@echo "Usage: \n"
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'

## fmt: go format
fmt:
	@echo "Gofmt..."
	@gofmt -w -l .

## test: run tests
test:
	@echo "Testing..."
	@go test -v ./...

## clean: clean up
clean:
	@echo "Cleaning..."
	@rm -rf ./bin

## build-n: build neutrino cli
build-n:
	@echo "Building neutrino cli..."
	@go build -v -o bin/neutrino ./cmd/neutrino

## build neutrino daemon
build-nd:
	@echo "Building neutrino daemon..."
	@export GO111MODULE=on; \
	env go build -tags netgo -ldflags="-s -w" -o bin/neutrinod ./cmd/neutrinod/main.go

## pg: starts postgres db inside docker container
pg:
	docker run --name neutrino-elements-pg -p 5432:5432 -e POSTGRES_USER=root -e POSTGRES_PASSWORD=secret -d postgres

## droppg: stop and remove postgres container
droppg:
	docker stop neutrino-elements-pg
	docker rm neutrino-elements-pg

## createdb: create db inside docker container
createdb:
	docker exec neutrino-elements-pg createdb --username=root --owner=root neutrino-elements

## dropdb: drops db inside docker container
dropdb:
	docker exec neutrino-elements-pg dropdb neutrino-elements

## createtestdb: create test db inside docker container
createtestdb:
	docker exec neutrino-elements-pg createdb --username=root --owner=root neutrino-elements-test

## droptestdb: drops test db inside docker container
droptestdb:
	docker exec neutrino-elements-pg dropdb neutrino-elements-test

## recreatedb: drop and create main and test db
recreatedb: dropdb createdb droptestdb createtestdb

## recreatetestdb: drop and create test db
recreatetestdb: droptestdb createtestdb

## pgcreatedb: starts docker container and creates dev and test db, used in CI
pgcreatedb:
	pg && createdb && createtestdb

## psql: connects to postgres terminal running inside docker container
psql:
	docker exec -it neutrino-elements-pg psql -U root -d neutrino-elements

## wpkh: creates el_wpkh wallet descriptor based on funded addresses pub_key
wpkh:
	go run ./script/fund_wpkh.go

## dev: run neutrinod and postgres
dev:
	export POSTGRES_USER=root; \
	export POSTGRES_PASSWORD=secret; \
	export POSTGRES_DB=neutrino-elements; \
	docker-compose up -d --build
