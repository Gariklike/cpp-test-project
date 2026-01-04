.PHONY: build run test docker-build docker-run clean

build:
	cd authorization-server && go build -o bin/authorization-server ./cmd/server

run: build
	cd authorization-server && ./bin/authorization-server

test:
	cd authorization-server && go test ./...

docker-build:
	docker build -t dnachulings/survey-auth-service:latest .

docker-run:
	docker run -p 8080:8080 --env-file .env dnachulings/survey-auth-service:latest

clean:
	rm -rf authorization-server/bin/