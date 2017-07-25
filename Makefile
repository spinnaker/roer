BINARY=roer

LDFLAGS="-X main.version=$(version)"

build: clean
	mkdir build
	env GOOS=darwin GOARCH=amd64 go build -ldflags ${LDFLAGS} -o build/${BINARY}-darwin-amd64 ./cmd/roer/main.go
	env GOOS=linux GOARCH=amd64 go build -ldflags ${LDFLAGS} -o build/${BINARY}-linux-amd64 ./cmd/roer/main.go
	env GOOS=linux GOARCH=386 go build -ldflags ${LDFLAGS} -o build/${BINARY}-linux-386 ./cmd/roer/main.go
	env GOOS=windows GOARCH=386 go build -ldflags ${LDFLAGS} -o build/${BINARY}-windows-386 ./cmd/roer/main.go
	env GOOS=windows GOARCH=amd64 go build -ldflags ${LDFLAGS} -o build/${BINARY}-windows-amd64 ./cmd/roer/main.go

package:
	cd build
	find . -name '${BINARY}-*' -print -exec zip '{}'.zip '{}' \;

clean:
	rm -rf build
