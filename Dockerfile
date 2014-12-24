FROM golang:1.3.3
MAINTAINER Alexis Montagne <alexis.montagne@gmail.com>

EXPOSE 8080

COPY . /go/app

ENV GOPATH=/go/app/Godeps/_workspace

CMD go run /go/app/fleet-ship.go
