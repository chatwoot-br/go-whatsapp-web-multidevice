# Use the official Go dev container as base
FROM mcr.microsoft.com/devcontainers/go:1-1.24-bookworm

# Install system dependencies
RUN apt-get update && apt-get install -y \
    ffmpeg \
    supervisor \
    curl \
    sudo \
    && rm -rf /var/lib/apt/lists/*

# Create necessary directories for supervisor and instances
RUN mkdir -p /etc/supervisor/conf.d \
    && mkdir -p /var/log/supervisor \
    && mkdir -p /app/instances \
    && mkdir -p /usr/local/bin \
    && mkdir -p /run

# Copy supervisord configuration
COPY .devcontainer/supervisord.conf /etc/supervisor/supervisord.conf

# Create a script to build and install the whatsapp binary
COPY .devcontainer/setup.sh /usr/local/bin/setup.sh
RUN chmod 755 /usr/local/bin/setup.sh

# Set proper permissions
RUN chown -R vscode:vscode /app/instances \
    && chown -R vscode:vscode /var/log/supervisor \
    && chown -R vscode:vscode /etc/supervisor/conf.d \
    && chmod 755 /app/instances \
    && chmod 755 /var/log/supervisor \
    && chmod 755 /etc/supervisor/conf.d

# Allow vscode user to run supervisor commands without password
RUN echo "vscode ALL=(ALL) NOPASSWD: /usr/bin/supervisord, /usr/bin/supervisorctl" >> /etc/sudoers

# Set working directory
WORKDIR /workspaces/go-whatsapp-web-multidevice
