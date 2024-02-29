B=$(shell git rev-parse --abbrev-ref HEAD)
BRANCH=$(subst /,-,$(B))
GITREV=$(shell git describe --abbrev=7 --always --tags)
REV=$(GITREV)-$(BRANCH)-$(shell date +%Y%m%d-%H:%M:%S)

info:
	- @echo "revision $(REV)"

build: info
	@ echo
	@ echo "Compiling Binary"
	@ echo
	# GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "-X main.revision=$(REV) -s -w" -o bin/gophermart app/main.go
	cd cmd/gophermart && go build -buildvcs=false -o gophermart

clean:
	@ echo
	@ echo "Cleaning"
	@ echo
	# rm bin/gophermart
	rm cmd/gophermart/gophermart

tidy:
	@ echo
	@ echo "Tidying"
	@ echo
	go mod tidy

run:
	go run cmd/gophermart/main.go -d "host=localhost port=5432 user=postgres dbname=postgres password=postgres sslmode=disable"

accrual:
	cmd/accrual/accrual_linux_amd64 -a :8081 -d "host=localhost port=5432 user=postgres dbname=postgres password=postgres sslmode=disable"

.PHONY: all build test clean tidy run accrual
