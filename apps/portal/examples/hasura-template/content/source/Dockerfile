FROM hasura/graphql-engine:v2.33.0.cli-migrations-v3

# Create the migrations directory and copy our initial SQL file
RUN mkdir -p /hasura-migrations
COPY migrations/ /hasura-migrations/

# TEMPORARILY COMMENT THIS OUT FOR THE TEMPLATE
# Developers will uncomment this after they use the Hasura CLI to generate metadata
# RUN mkdir -p /hasura-metadata
# COPY metadata/ /hasura-metadata/

HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:8080/healthz || exit 1

EXPOSE 8080