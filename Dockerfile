FROM golang:1.15-alpine3.12

RUN mkdir /app
ADD . /app
WORKDIR /app

RUN go build .

EXPOSE 8080
CMD ["./universal-studios"]
