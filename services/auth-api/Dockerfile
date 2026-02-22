FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY pkg/ pkg/
COPY services/auth-api/ services/auth-api/
RUN CGO_ENABLED=0 go build -o /auth-api ./services/auth-api

FROM alpine:3
RUN apk add --no-cache ca-certificates
COPY --from=build /auth-api /auth-api
EXPOSE 8081
ENTRYPOINT ["/auth-api"]
