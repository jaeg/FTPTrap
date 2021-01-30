# Go parameters
 GOCMD=go
 GOBUILD=$(GOCMD) build
 GOCLEAN=$(GOCMD) clean
 GOTEST=$(GOCMD) test
 GOGET=$(GOCMD) get
 BINARY_NAME=FTPTrap
 BINARY_UNIX=$(BINARY_NAME)_unix
 REPO_NAME=jaeg/ftptrap

 all: test build
 build:
				 $(GOBUILD) -o ./bin/$(BINARY_NAME) -v
 build-linux:
				CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -o ./bin/$(BINARY_UNIX) -v
 test:
				 $(GOTEST) -v ./...
 clean:
				 $(GOCLEAN)
				 rm -f ./bin/$(BINARY_NAME)
 run: build
	./bin/$(BINARY_NAME) --key-path test.key
image: build-linux
	docker build ./ -t $(REPO_NAME):latest
	docker tag $(REPO_NAME):latest $(REPO_NAME):$(shell git describe --abbrev=0 --tags)-$(shell git rev-parse --short HEAD)
publish:
	docker push $(REPO_NAME):latest
	docker push $(REPO_NAME):$(shell git describe --abbrev=0 --tags)-$(shell git rev-parse --short HEAD)
release:
	docker tag $(REPO_NAME):$(shell git describe --abbrev=0 --tags)-$(shell git rev-parse --short HEAD) $(REPO_NAME):$(shell git describe --abbrev=0 --tags)
	docker push $(REPO_NAME):$(shell git describe --abbrev=0 --tags)
	docker push $(REPO_NAME):latest