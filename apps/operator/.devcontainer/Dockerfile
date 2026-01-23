FROM mcr.microsoft.com/devcontainers/go:2-1.25-bookworm

# Ensure vscode user exists with correct UID/GID
ARG USERNAME=vscode
ARG USER_UID=1000
ARG USER_GID=1000

# Install essential tools as root
USER root

# Install bash and other essential tools first
RUN apt-get update && apt-get install -y --no-install-recommends \
  bash \
  coreutils \
  git \
  ca-certificates \
  curl \
  procps \
  lsb-release \
  && rm -rf /var/lib/apt/lists/*

# Ensure vscode user's cache directory is writable for VS Code extensions
RUN mkdir -p /home/$USERNAME/.cache \
  && chown -R $USERNAME:$USERNAME /home/$USERNAME/.cache

# The base image already has vscode user, just ensure correct UID/GID
RUN if [ "$USER_GID" != "1000" ] || [ "$USER_UID" != "1000" ]; then \
  groupmod --gid $USER_GID $USERNAME \
  && usermod --uid $USER_UID --gid $USER_GID $USERNAME \
  && chown -R $USER_UID:$USER_GID /home/$USERNAME; \
  fi

# Set bash as the default shell
ENV SHELL=/bin/bash
ENV DEBIAN_FRONTEND=noninteractive

# Set the default user
USER $USERNAME

# Keep the container running for VS Code Dev Containers
CMD ["sleep", "infinity"]
