FROM golang:1.23.1

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY *.go ./

RUN GOOS=linux go build -o ./app

RUN mkdir /app/datas
CMD ["/app/app"]
