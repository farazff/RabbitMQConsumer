FROM golang:1.17 AS build

WORKDIR /app

COPY . .

RUN go build -o /binary

## Deploy
FROM golang:1.17

WORKDIR /

COPY --from=build /binary /binary

EXPOSE 8080

ENTRYPOINT ["/binary"]
