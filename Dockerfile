FROM harbor.lab:8080/library/golang:1.23-alpine AS builder
WORKDIR /app

ENV GOPROXY=http://nexus.lab:8081/repository/go-proxy/,direct
ENV GONOSUMCHECK=*
ENV GOFLAGS=-insecure

COPY go.mod ./
RUN go mod tidy

COPY . .
RUN go mod tidy
RUN go build -o corona-go .

FROM harbor.lab:8080/library/alpine:3.19
WORKDIR /app
COPY --from=builder /app/corona-go .

RUN adduser -D -u 1000 appuser
USER appuser

EXPOSE 8080
CMD ["./corona-go"]
