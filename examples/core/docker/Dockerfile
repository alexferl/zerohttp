FROM golang:1.25-alpine AS builder

WORKDIR /build

RUN apk add --no-cache git
RUN adduser -D -u 1337 zerohttp

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG GOOS=linux
ARG GOARCH=amd64

ENV GOOS=$GOOS
ENV GOARCH=$GOARCH

RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o zerohttp ./main.go

FROM scratch
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /build/zerohttp /zerohttp

USER zerohttp

ENTRYPOINT ["/zerohttp"]
