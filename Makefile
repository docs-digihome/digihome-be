include .env
export

.PHONY: run docker-build

# run app with go run command
run:
	@go run cmd/main.go

# build app docker image
docker-build:
	@./script/docker-build.sh

dev-start:
	@docker compose up -d

# migrations

migrate-up:
	@migrate -path migrations/ -database "postgresql://${DB_USER}:${DB_PASSWORD}@localhost:5432/${DB_NAME}?sslmode=disable" -verbose up

migrate-down:
	@migrate -path migrations/ -database "postgresql://${DB_USER}:${DB_PASSWORD}@localhost:5432/${DB_NAME}?sslmode=disable" -verbose down

migrate-status:
	@migrate -path migrations/ -database "postgresql://${DB_USER}:${DB_PASSWORD}@localhost:5432/${DB_NAME}?sslmode=disable" version

# genereate self sign certificate for tls for 1 year
cert-gen:
	@mkdir -p ./config/cert
	@openssl req -x509 -newkey rsa:4096 -keyout ./config/cert/server.key -out ./config/cert/server.crt -days 365 -nodes -subj "/C=ID/ST=Jakarta/L=Jakarta/O=YourOrg/OU=IT/CN=localhost"

# build binary for all os and arch
build-binary:
	@./script/binary-build.sh
