FROM golang:1.15-rc
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

FROM alpine:3.12.0
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=0 /app/main ./
ENTRYPOINT ./main
