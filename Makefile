
# 0.0 shouldn't clobber any released builds
TAG = latest
PREFIX = registry.dstresearch.com/cron
APPNAME = cron

.PHONY: all build restore

all: container

container: build
	upx -9 $(APPNAME)
	godep go test
	#godep go test ./pkg/...
	docker build -t $(PREFIX):$(TAG) .

build: restore
	godep go vet
	golint
	CGO_ENABLED=0 GOOS=linux godep go build -a -ldflags '-s'
	#godep go build

push:
	docker push $(PREFIX):$(TAG)

more_cool_tests:
	go test -bench-. -cpu=1,4,16 -benchmem

escape_analysis:
	go test -gcflags=-m -bench=something
	# see https://golang.org/pkg/net/http/pprof/
	go tool pprof http://localhost:9090/debug/pprof/profile
	go tool pprof http://localhost:9090/debug/pprof/heap

deploy:
	go test -tags=integration

restore: Godeps/_workspace/src/github.com/aws/aws-sdk-go/NOTICE.txt

Godeps/_workspace/src/github.com/aws/aws-sdk-go/NOTICE.txt:
	godep restore
