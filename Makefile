prepare-env:
	@if [ ! -f ./.env ]; then cp ./.env.example ./.env; fi

start: prepare-env
	docker compose -f deploy/docker-compose.yaml up --build
kill:
	docker compose -f deploy/docker-compose.yaml down -v

unittest:
	go test ./...

e2e:
	docker-compose -f deploy/docker-compose.e2e.yaml up --build --abort-on-container-exit 

tidy:
	@go mod tidy

lint:
	@golangci-lint run
