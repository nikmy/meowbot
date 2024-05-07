FROM golang:1.22.1-alpine

WORKDIR /app

ADD . .

RUN go build ./cmd/meowbot/
RUN rm go.mod

EXPOSE 8080
ENTRYPOINT [ "./server" ]
