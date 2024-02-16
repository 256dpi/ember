all:
	go fmt ./...
	go vet ./...
	golint ./...

install:
	go install ./ember-serve

debug:
	go run ./ember-serve -fastboot -timeout 200ms ./example/dist

test:
	vegeta attack -duration=30s -rate=1000 -max-workers=5 -targets=vegeta.cfg | vegeta report -every=1s
