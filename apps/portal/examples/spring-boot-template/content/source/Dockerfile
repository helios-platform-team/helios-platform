# ──────────────────────────────────────────────────────────
# Multi-stage Dockerfile for Spring Boot (Gradle)
# ──────────────────────────────────────────────────────────
# Stage 1 — Build
#   Uses the official Gradle image with JDK 21.  We copy sources
#   into the container and run a full Gradle build.  The resulting
#   fat JAR is extracted for the next stage.
#
# Stage 2 — Runtime
#   Uses a minimal Eclipse Temurin JRE-only Alpine image to keep
#   the final image small (~200 MB vs ~800 MB with a full JDK).
#
# Kaniko Compatibility
#   This Dockerfile avoids features that are problematic for
#   Kaniko (e.g. mounting secrets, multi-platform builds).
#   The build is fully self-contained.
# ──────────────────────────────────────────────────────────

# ---- Build Stage ----
FROM gradle:8.13-jdk21 AS builder

WORKDIR /workspace

# Copy Gradle wrapper & build scripts first to leverage Docker
# layer caching — dependencies are only re-downloaded when these
# files change.
COPY build.gradle settings.gradle ./
COPY gradle ./gradle/

# Download dependencies in a separate layer for caching
RUN gradle dependencies --no-daemon || true

# Copy source code and build
COPY src ./src/
RUN gradle bootJar --no-daemon -x test

# ---- Production Stage ----
FROM eclipse-temurin:21-jre-alpine AS production

# Create a non-root user for security
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

# Copy the fat JAR from the builder stage
COPY --from=builder /workspace/build/libs/app.jar ./app.jar

# Switch to non-root user
USER appuser

EXPOSE ${{ values.port }}

# Use exec form so the JVM receives signals (SIGTERM) correctly
# for graceful shutdown in Kubernetes.
ENTRYPOINT ["java", \
    "-XX:+UseContainerSupport", \
    "-XX:MaxRAMPercentage=75.0", \
    "-Djava.security.egd=file:/dev/./urandom", \
    "-jar", "app.jar"]
