FROM golang:1.24-alpine AS builder
ENV GOOS=linux \
    GOARCH=amd64

WORKDIR /build

COPY . .
RUN go mod tidy
RUN go build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o ticketing-go-img main.go


#############################
# CREATE the runtime 
#############################
FROM alpine:3
WORKDIR /app
# Create nonroot user for runtime
ENV USER=appuser
ENV UID=10001 
ENV TZ=Asia/Jakarta

# See https://stackoverflow.com/a/55757473/12429735RUN 
RUN adduser \    
    --disabled-password \    
    --gecos "" \    
    --home "/nonexistent" \    
    --shell "/sbin/nologin" \    
    --no-create-home \    
    --uid "${UID}" \    
    "${USER}" && \
    apk add tzdata

RUN echo "Asia/Jakarta" > /etc/timezone

COPY --from=builder /build/ticketing-go-img /app/

RUN apk --no-cache add ca-certificates
RUN chown -R appuser:appuser /app

# Use an unprivileged user.
USER appuser:appuser

EXPOSE 3000

HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 CMD curl -f http://localhost:3000/health || exit 1
CMD ["/app/monika-go-img"]
