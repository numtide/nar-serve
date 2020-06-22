FROM golang:1.14 as builder

WORKDIR /go/src/app
COPY go.mod go.sum ./

RUN go mod download

COPY . ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -ldflags '-w -extldflags "-static"' -o /go-webserver ./*.go

FROM alpine
RUN apk add --no-cache ca-certificates

COPY --from=builder /go-webserver /app

ENTRYPOINT ["/app"]
