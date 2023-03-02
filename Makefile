ll: test

test:
	go mod tidy
	go test -v -race .
