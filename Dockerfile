# Build stage — uses Nexus go-proxy (anonymous reads)
FROM harbor.lab:8080/library/golang:1.26.4-alpine AS build
WORKDIR /build

# Route module fetches through Nexus
ENV GOPROXY=http://nexus.lab:8081/repository/go-proxy/,direct
ENV GOSUMDB=off

COPY go.mod go.sum ./
RUN go mod download

COPY . .
ARG BUILD_VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -o /out/app .

# Runtime stage — minimal image
FROM harbor.lab:8080/library/alpine:3.19
RUN adduser -D -u 1000 appuser
USER appuser
COPY --from=build /out/app /usr/local/bin/app
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/app"]

