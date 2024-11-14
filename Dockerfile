FROM golang:1.23.3-alpine
COPY main.go .
CMD ["go", "run", "main.go"]
