all:
	@go mod tidy
	@go build -o ./grpc-server server/main.go
	@go build -o ./grpc-client client/main.go

clean cl:
	rm -f grpc-server grpc-client
