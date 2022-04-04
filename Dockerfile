FROM golang:alpine as build

WORKDIR /app
COPY . .
RUN go build -o bin/server -buildvcs=false

FROM alpine AS app

ENV PORT=8080

WORKDIR /app

COPY --from=build /app/bin/server /app/bin/

EXPOSE 8080
CMD ["/app/bin/server"]
