FROM golang:1.23.1

WORKDIR /app

COPY . .

RUN go mod download
RUN GOOS=linux go build -o ./app

RUN mkdir /app/datas
CMD ["/app/app"]
