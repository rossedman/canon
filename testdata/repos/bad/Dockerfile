FROM golang:1.27 AS build
WORKDIR /src
COPY . .
RUN go build -o /app .

FROM debian:bookworm
RUN apt-get update && apt-get install -y curl
COPY --from=build /app /app
ENTRYPOINT ["/app"]
