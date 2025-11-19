# Multi-stage build for Aurora Workload Simulator
FROM maven:3.9-amazoncorretto-17 AS builder

WORKDIR /app

# Copy Maven configuration
COPY pom.xml .

# Download dependencies (cached layer)
RUN mvn dependency:go-offline -B

# Copy source code
COPY src ./src

# Build application
RUN mvn clean package -DskipTests

# Runtime stage
FROM amazoncorretto:17-alpine

WORKDIR /app

# Copy the built JAR from builder stage
COPY --from=builder /app/target/workload-simulator-1.0.0.jar workload-simulator.jar

# Create non-root user for security
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser && \
    chown -R appuser:appuser /app

USER appuser

# Expose Prometheus metrics port
EXPOSE 8080

# Set JVM options for container environment
ENV JAVA_OPTS="-XX:+UseContainerSupport \
               -XX:MaxRAMPercentage=75.0 \
               -XX:+UseG1GC \
               -XX:+HeapDumpOnOutOfMemoryError \
               -XX:HeapDumpPath=/app/heapdump.hprof"

# Default environment variables (can be overridden)
ENV DB_PASSWORD=""
ENV AURORA_ENDPOINT=""
ENV DATABASE_NAME="lab_db"
ENV USERNAME="admin"
ENV WRITE_WORKERS="10"
ENV WRITE_RATE="100"
ENV CONNECTION_POOL_SIZE="100"
ENV LOG_INTERVAL="10"
ENV ENABLE_METRICS="true"

# Entrypoint script
ENTRYPOINT ["sh", "-c", "java $JAVA_OPTS -jar workload-simulator.jar \
    --aurora-endpoint ${AURORA_ENDPOINT} \
    --database-name ${DATABASE_NAME} \
    --username ${USERNAME} \
    --password ${DB_PASSWORD} \
    --write-workers ${WRITE_WORKERS} \
    --write-rate ${WRITE_RATE} \
    --connection-pool-size ${CONNECTION_POOL_SIZE} \
    --log-interval ${LOG_INTERVAL} \
    ${ENABLE_METRICS:+--enable-metrics}"]

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=60s --retries=3 \
    CMD wget --quiet --tries=1 --spider http://localhost:8080/metrics || exit 1
