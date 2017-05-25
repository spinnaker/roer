BINARY=roer

VERSION=`git rev-parse HEAD`

LDFLAGS="-X main.Version=${VERSION}"

build:
	mkdir build
	env GOOS=darwin GOARCH=amd64 go build -o build/${BINARY}-darwin-amd64 .
	env GOOS=linux GOARCH=amd64 go build -o build/${BINARY}-linux-amd64 .
	env GOOS=linux GOARCH=386 go build -o build/${BINARY}-linux-386 .
	env GOOS=windows GOARCH=386 go build -o build/${BINARY}-windows-386 .
	env GOOS=windows GOARCH=amd64 go build -o build/${BINARY}-windows-amd64 .

clean:
	rm -rf build
