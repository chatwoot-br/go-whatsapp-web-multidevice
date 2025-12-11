############################
# STEP 1 build executable binary
############################
FROM golang:1.24-alpine3.20 AS builder
RUN apk update && apk add --no-cache gcc musl-dev gcompat
WORKDIR /whatsapp
COPY ./src .

# Fetch dependencies.
RUN go mod download
# Build the binary with optimizations
RUN go build -a -ldflags="-w -s" -o /app/whatsapp

#############################
## STEP 2 build a smaller image
#############################
FROM alpine:3.20
# Install runtime dependencies including mailcap for MIME types database (/etc/mime.types)
# mailcap provides proper MIME type -> file extension mapping for media files
RUN apk add --no-cache ffmpeg supervisor curl python3 py3-pip net-tools mailcap tzdata
ENV TZ=UTC
WORKDIR /app
# Copy compiled from builder.
COPY --from=builder /app/whatsapp /app/whatsapp

# Copy startup script
COPY docker/start-admin.sh /app/start-admin.sh
RUN chmod +x /app/start-admin.sh

# Create necessary directories for supervisor and instances
RUN mkdir -p /etc/supervisor/conf.d /var/log/supervisor /app/instances /run

# Copy the correct supervisord configuration
COPY .devcontainer/supervisord.conf /etc/supervisor/supervisord.conf

# Make the whatsapp binary available globally
RUN ln -s /app/whatsapp /usr/local/bin/whatsapp

# Run the binary.
ENTRYPOINT ["/app/whatsapp"]

CMD [ "rest" ]