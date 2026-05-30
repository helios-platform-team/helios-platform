# ---- Build Stage ----
FROM mcr.microsoft.com/dotnet/sdk:10.0.300-alpine3.23 AS build
WORKDIR /src

# Copy project files and restore dependencies (maximizes layer caching)
COPY DotNetApi.csproj ./
RUN dotnet restore DotNetApi.csproj

# Copy source and publish
COPY . .
RUN dotnet publish DotNetApi.csproj -c Release -o /app/publish /p:UseAppHost=false

# ---- Runtime Stage ----
FROM mcr.microsoft.com/dotnet/aspnet:10.0.8-alpine3.23 AS runtime
WORKDIR /app

# Copy published files from build stage with correct ownership
# Note: aspnet images starting from .NET 8 have a built-in non-root user 'app' (UID 1654, GID 1654)
COPY --from=build --chown=app:app /app/publish .

# Set environment variables
ENV ASPNETCORE_URLS=http://+:${{ values.port }} \
  ASPNETCORE_ENVIRONMENT=Production

# Labels for metadata
LABEL org.opencontainers.image.source="https://github.com/helios-platform-team/helios-platform" \
  org.opencontainers.image.description="Helios .NET Template API Application" \
  org.opencontainers.image.licenses="MIT"

# Security: Switch to non-root user
USER app

EXPOSE ${{ values.port }}

# Health check using busybox wget (built-in on Alpine)
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:${{ values.port }}/health || exit 1

ENTRYPOINT ["dotnet", "DotNetApi.dll"]
