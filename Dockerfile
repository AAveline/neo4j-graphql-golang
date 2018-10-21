
FROM golang:1.8

WORKDIR /go/src/graphql

COPY . .

RUN go get .

RUN go get github.com/graphql-go/graphql

