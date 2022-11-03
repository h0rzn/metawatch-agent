FROM golang:1.19

WORKDIR /
COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .
RUN go build -o /usr/local/bin/app ./..
EXPOSE 8080
CMD ["app"]
