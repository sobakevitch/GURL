APP?=gurl
PROJECT?=hatter/gurl
GOOS?=linux
GOARCH?=amd64

all: build windows

clean:
	/bin/rm -f ${APP} ${APP}.exe

build: clean
	CGO_ENABLED=0 GOOS=${GOOS} GOARCH=${GOARCH} go build -o ${APP}
	/bin/mv ${APP} ${GOPATH}/bin


test:
	go test -v -race ./...

windows: clean
	CGO_ENABLED=0 GOOS=windows GOARCH=${GOARCH} go build -o ${APP}.exe
	
