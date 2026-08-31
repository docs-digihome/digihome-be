FROM golang:1.25.8-alpine3.23 AS builder
WORKDIR /app
COPY . .

RUN apk update && apk add upx
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o "./bin/dist/backend" ./cmd

RUN upx --best --lzma ./bin/dist/backend

FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
	ocrmypdf \
	tesseract-ocr \
	tesseract-ocr-eng \
	tesseract-ocr-ind \
	tesseract-ocr-osd \
	ghostscript \
	qpdf \
	&& rm -rf /var/lib/apt/lists/*

COPY --from=builder /app/bin/dist/backend /

ENTRYPOINT ["/backend"]
