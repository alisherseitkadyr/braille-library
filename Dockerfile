FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# api is the server; seed and recompute are used by the "init" service in docker-compose.yml.
RUN CGO_ENABLED=0 go build -o /out/api ./cmd/api && \
    CGO_ENABLED=0 go build -o /out/seed ./cmd/seed && \
    CGO_ENABLED=0 go build -o /out/recompute ./cmd/recompute

FROM alpine:3.21
RUN adduser -D -H app
USER app
COPY --from=build /out/ /usr/local/bin/
EXPOSE 8080
ENTRYPOINT ["api"]
