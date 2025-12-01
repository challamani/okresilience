FROM golang:1.22-alpine AS build
WORKDIR /app
COPY server.go .
RUN go build -o tcp-reset-service server.go

FROM alpine:3.19
RUN adduser -D app
USER app
COPY --from=build /app/tcp-reset-service /tcp-reset-service
EXPOSE 80
ENTRYPOINT ["/tcp-reset-service"]
