FROM golang:buster

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY *.go ./

RUN go build -o /gobot-ci

EXPOSE 8080

CMD [ "/gobot-ci" ]