FROM golang:1.21-alpine

WORKDIR /app
COPY web/go.mod web/go.sum ./
RUN go mod download

COPY web/ .

RUN go build -o messenger .

EXPOSE 8080
CMD ["./messenger"]
