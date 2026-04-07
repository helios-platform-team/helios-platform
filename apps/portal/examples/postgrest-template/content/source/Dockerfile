# Use official PostgREST image
FROM postgrest/postgrest:v12.2.0

# Copy configuration file (will be injected by Helios Operator at runtime)
# COPY postgrestrc.conf /etc/postgrest/postgrestrc.conf

# The database connection URI will be provided via environment variable
# PGRST_DB_URI injected by the Helios Operator

# Health check
HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:3000/ || exit 1

EXPOSE 3000
