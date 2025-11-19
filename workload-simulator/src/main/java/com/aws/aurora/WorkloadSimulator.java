package com.aws.aurora;

import com.zaxxer.hikari.HikariConfig;
import com.zaxxer.hikari.HikariDataSource;
import io.prometheus.client.Counter;
import io.prometheus.client.Histogram;
import io.prometheus.client.exporter.HTTPServer;
import io.prometheus.client.hotspot.DefaultExports;
import org.apache.commons.cli.*;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import javax.sql.DataSource;
import java.io.IOException;
import java.sql.Connection;
import java.sql.PreparedStatement;
import java.sql.SQLException;
import java.time.LocalDateTime;
import java.time.format.DateTimeFormatter;
import java.util.ArrayList;
import java.util.List;
import java.util.Random;
import java.util.concurrent.*;
import java.util.concurrent.atomic.AtomicLong;

/**
 * Aurora Blue-Green Deployment Workload Simulator
 * Generates write workload to test Aurora cluster behavior during Blue-Green switchover
 */
public class WorkloadSimulator {
    private static final Logger logger = LoggerFactory.getLogger(WorkloadSimulator.class);
    private static final DateTimeFormatter timeFormatter = DateTimeFormatter.ofPattern("yyyy-MM-dd HH:mm:ss.SSS");

    // Configuration
    private final String auroraEndpoint;
    private final String databaseName;
    private final String username;
    private final String password;
    private final int writeWorkers;
    private final int writeRate;
    private final int connectionPoolSize;
    private final int logInterval;
    private final boolean enableMetrics;

    // Resources
    private DataSource dataSource;
    private ExecutorService executorService;
    private ScheduledExecutorService scheduledExecutor;
    private HTTPServer prometheusServer;

    // Statistics
    private final AtomicLong totalRequests = new AtomicLong(0);
    private final AtomicLong successfulRequests = new AtomicLong(0);
    private final AtomicLong failedRequests = new AtomicLong(0);

    // Prometheus Metrics
    private static final Counter writeRequests = Counter.build()
            .name("aurora_write_requests_total")
            .help("Total write requests")
            .labelNames("status")
            .register();

    private static final Histogram writeLatency = Histogram.build()
            .name("aurora_write_latency_seconds")
            .help("Write operation latency in seconds")
            .buckets(0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0)
            .register();

    private static final Counter connectionErrors = Counter.build()
            .name("aurora_connection_errors_total")
            .help("Total connection errors")
            .labelNames("error_type")
            .register();

    public WorkloadSimulator(String auroraEndpoint, String databaseName, String username, String password,
                            int writeWorkers, int writeRate, int connectionPoolSize, int logInterval,
                            boolean enableMetrics) {
        this.auroraEndpoint = auroraEndpoint;
        this.databaseName = databaseName;
        this.username = username;
        this.password = password;
        this.writeWorkers = writeWorkers;
        this.writeRate = writeRate;
        this.connectionPoolSize = connectionPoolSize;
        this.logInterval = logInterval;
        this.enableMetrics = enableMetrics;
    }

    /**
     * Initialize database connection pool with AWS JDBC Wrapper
     */
    private void initializeDataSource() throws SQLException {
        logger.info("Initializing HikariCP connection pool...");

        HikariConfig config = new HikariConfig();

        // AWS Advanced JDBC Wrapper configuration
        // Format: jdbc:aws-wrapper:mysql://endpoint:port/database
        String jdbcUrl = String.format("jdbc:aws-wrapper:mysql://%s:3306/%s", auroraEndpoint, databaseName);
        config.setJdbcUrl(jdbcUrl);
        config.setUsername(username);
        config.setPassword(password);

        // HikariCP pool settings
        config.setMaximumPoolSize(connectionPoolSize);
        config.setMinimumIdle(Math.min(10, connectionPoolSize));
        config.setConnectionTimeout(30000); // 30 seconds
        config.setIdleTimeout(600000); // 10 minutes
        config.setMaxLifetime(1800000); // 30 minutes
        config.setLeakDetectionThreshold(0); // Disable leak detection to avoid false alarms during failover

        // AWS JDBC Wrapper specific properties
        // Blue-Green plugin: Proactively monitors Blue-Green deployment status for minimal downtime
        // Failover plugin: Handles general cluster failover scenarios
        // EFM plugin: Enhanced Failure Monitoring for proactive connection health checks
        config.addDataSourceProperty("wrapperPlugins", "bg,failover,efm");

        // AWS JDBC Wrapper logging - FINEST level for detailed Blue-Green plugin activity
        config.addDataSourceProperty("wrapperLoggerLevel", "FINEST");

        // Blue-Green plugin configuration
        config.addDataSourceProperty("bgdId", "1"); // Blue-Green Deployment ID (required for bg plugin)
        config.addDataSourceProperty("bgConnectTimeoutMs", "30000"); // 30 seconds - max wait for new connections during switchover
        config.addDataSourceProperty("bgSwitchoverTimeoutMs", "180000"); // 3 minutes - max switchover duration

        // Failover plugin configuration
        config.addDataSourceProperty("failoverTimeoutMs", "10000"); // 10 seconds - aggressive fail-fast for minimal downtime
        config.addDataSourceProperty("failoverClusterTopologyRefreshRateMs", "1000"); // 1 second - faster topology detection
        config.addDataSourceProperty("enableClusterAwareFailover", "true");
        config.addDataSourceProperty("clusterInstanceHostPattern", "?.cluster-?.us-east-1.rds.amazonaws.com");

        // MySQL specific settings
        config.addDataSourceProperty("cachePrepStmts", "true");
        config.addDataSourceProperty("prepStmtCacheSize", "250");
        config.addDataSourceProperty("prepStmtCacheSqlLimit", "2048");
        config.addDataSourceProperty("useServerPrepStmts", "true");
        config.addDataSourceProperty("useLocalSessionState", "true");
        config.addDataSourceProperty("rewriteBatchedStatements", "true");
        config.addDataSourceProperty("cacheResultSetMetadata", "true");
        config.addDataSourceProperty("elideSetAutoCommits", "true");
        config.addDataSourceProperty("maintainTimeStats", "false");

        this.dataSource = new HikariDataSource(config);
        logger.info("Connection pool initialized successfully");
    }

    /**
     * Start Prometheus metrics server (for Kubernetes deployment)
     */
    private void startMetricsServer() throws IOException {
        if (enableMetrics) {
            DefaultExports.initialize();
            prometheusServer = new HTTPServer(8080);
            logger.info("Prometheus metrics server started on port 8080");
        }
    }

    /**
     * Initialize and start the workload simulator
     */
    public void start() throws Exception {
        logBanner();
        logConfiguration();

        // Initialize resources
        initializeDataSource();
        startMetricsServer();

        // Create thread pool for workers
        executorService = Executors.newFixedThreadPool(writeWorkers);
        scheduledExecutor = Executors.newScheduledThreadPool(2);

        // Schedule statistics logging
        scheduledExecutor.scheduleAtFixedRate(this::logStatistics, logInterval, logInterval, TimeUnit.SECONDS);

        // Start write workers
        logger.info("Starting {} write workers...", writeWorkers);
        List<Future<?>> workerFutures = new ArrayList<>();
        for (int i = 1; i <= writeWorkers; i++) {
            Future<?> future = executorService.submit(new WriteWorker(i));
            workerFutures.add(future);
        }

        // Wait for shutdown signal
        Runtime.getRuntime().addShutdownHook(new Thread(this::shutdown));

        // Keep main thread alive
        try {
            for (Future<?> future : workerFutures) {
                future.get();
            }
        } catch (InterruptedException | ExecutionException e) {
            logger.error("Worker execution interrupted", e);
        }
    }

    /**
     * Shutdown the simulator gracefully
     */
    private void shutdown() {
        logger.info("Shutting down workload simulator...");

        if (executorService != null) {
            executorService.shutdownNow();
        }
        if (scheduledExecutor != null) {
            scheduledExecutor.shutdownNow();
        }
        if (dataSource instanceof HikariDataSource) {
            ((HikariDataSource) dataSource).close();
        }
        if (prometheusServer != null) {
            prometheusServer.close();
        }

        logFinalStatistics();
        logger.info("Workload simulator stopped");
    }

    /**
     * Write worker thread - executes continuous write operations
     */
    private class WriteWorker implements Runnable {
        private final int workerId;
        private final Random random = new Random();
        private final int delayMs;
        private String lastKnownHost = null;

        public WriteWorker(int workerId) {
            this.workerId = workerId;
            // Calculate delay to achieve target write rate
            this.delayMs = writeRate > 0 ? 1000 / writeRate : 100;
        }

        @Override
        public void run() {
            logger.info("Worker-{} started", workerId);

            while (!Thread.currentThread().isInterrupted()) {
                try {
                    executeWrite();

                    // Rate limiting
                    if (delayMs > 0) {
                        Thread.sleep(delayMs);
                    }
                } catch (InterruptedException e) {
                    Thread.currentThread().interrupt();
                    break;
                } catch (Exception e) {
                    // Continue running even on errors
                    logger.debug("Worker-{} encountered error: {}", workerId, e.getMessage());
                }
            }

            logger.info("Worker-{} stopped", workerId);
        }

        /**
         * Execute a single write operation with retry logic
         */
        private void executeWrite() {
            String tableName = String.format("test_%04d", random.nextInt(12000) + 1);
            int maxRetries = 5; // Increased retries for minimal downtime
            int retryDelayMs = 500; // Start with 500ms - faster retry for minimal downtime

            for (int attempt = 1; attempt <= maxRetries; attempt++) {
                long startTime = System.nanoTime();

                try (Connection conn = dataSource.getConnection();
                     PreparedStatement stmt = conn.prepareStatement(
                         "INSERT INTO " + tableName + " (col1, col2, col3, col4, col5) VALUES (?, ?, ?, ?, ?)")) {

                    // Generate random data
                    stmt.setString(1, generateRandomString(20));
                    stmt.setInt(2, random.nextInt(1000));
                    stmt.setString(3, generateRandomString(50));
                    stmt.setDouble(4, random.nextDouble() * 1000);
                    stmt.setLong(5, System.currentTimeMillis());

                    stmt.executeUpdate();

                    long latencyNanos = System.nanoTime() - startTime;
                    double latencyMs = latencyNanos / 1_000_000.0;

                    // Get current connection info
                    String currentHost = getCurrentHost(conn);

                    // Detect host switch (indicates Blue-Green switchover or failover)
                    if (lastKnownHost != null && !currentHost.equals(lastKnownHost)) {
                        logger.info("[{}] INFO: Worker-{} | Switched to new host: {} (from: {})",
                                getCurrentTime(), workerId, currentHost, lastKnownHost);
                    }
                    lastKnownHost = currentHost;

                    successfulRequests.incrementAndGet();
                    totalRequests.incrementAndGet();
                    writeRequests.labels("success").inc();
                    writeLatency.observe(latencyNanos / 1_000_000_000.0);

                    logger.debug("[{}] SUCCESS: Worker-{} | Host: {} | Table: {} | INSERT completed | Latency: {}ms{}",
                            getCurrentTime(), workerId, currentHost, tableName, String.format("%.2f", latencyMs),
                            attempt > 1 ? " (retry " + (attempt - 1) + ")" : "");

                    return; // Success - exit retry loop

                } catch (SQLException e) {
                    String errorType = categorizeError(e);
                    boolean isFailoverError = errorType.contains("connection") || errorType.contains("failover");

                    if (attempt < maxRetries && isFailoverError) {
                        // Retry for connection/failover errors
                        logger.warn("[{}] ERROR: Worker-{} | Table: {} | {} | Retry {}/{} in {}ms | Error: {}",
                                getCurrentTime(), workerId, tableName, errorType, attempt, maxRetries,
                                retryDelayMs, e.getMessage());

                        try {
                            Thread.sleep(retryDelayMs);
                            retryDelayMs *= 2; // Exponential backoff
                        } catch (InterruptedException ie) {
                            Thread.currentThread().interrupt();
                            break;
                        }
                    } else {
                        // Final failure or non-retryable error
                        failedRequests.incrementAndGet();
                        totalRequests.incrementAndGet();
                        writeRequests.labels("failure").inc();
                        connectionErrors.labels(errorType).inc();

                        logger.error("[{}] ERROR: Worker-{} | Table: {} | {} | Error: {}{}",
                                getCurrentTime(), workerId, tableName, errorType, e.getMessage(),
                                attempt > 1 ? " (after " + (attempt - 1) + " retries)" : "");

                        if (isFailoverError) {
                            logger.info("[{}] INFO: Worker-{} | Will retry on next operation...",
                                    getCurrentTime(), workerId);
                        }
                        break;
                    }
                }
            }
        }

        private String generateRandomString(int length) {
            String chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789";
            StringBuilder sb = new StringBuilder(length);
            for (int i = 0; i < length; i++) {
                sb.append(chars.charAt(random.nextInt(chars.length())));
            }
            return sb.toString();
        }

        private String categorizeError(SQLException e) {
            String message = e.getMessage().toLowerCase();
            if (message.contains("communications link failure") || message.contains("connection")) {
                return "connection_lost";
            } else if (message.contains("timeout")) {
                return "timeout";
            } else if (message.contains("deadlock")) {
                return "deadlock";
            } else {
                return "other";
            }
        }

        private String getCurrentHost(Connection conn) {
            try (PreparedStatement stmt = conn.prepareStatement("SELECT @@hostname, @@read_only");
                 java.sql.ResultSet rs = stmt.executeQuery()) {
                if (rs.next()) {
                    String hostname = rs.getString(1);
                    int readOnly = rs.getInt(2);

                    // Extract instance identifier from hostname
                    // Format: ip-10-0-1-123.ec2.internal or similar
                    // We'll return a shortened version with role indicator
                    if (hostname != null && !hostname.isEmpty()) {
                        // Try to extract meaningful part (first segment before first dot)
                        String shortHost = hostname;
                        int dotIndex = hostname.indexOf('.');
                        if (dotIndex > 0) {
                            shortHost = hostname.substring(0, dotIndex);
                        }

                        // Add role indicator (writer/reader)
                        String role = readOnly == 0 ? "writer" : "reader";
                        return shortHost + " (" + role + ")";
                    }
                }
            } catch (SQLException e) {
                // Silently handle - don't want to impact write performance
                logger.debug("Failed to get hostname: {}", e.getMessage());
            }
            return "unknown";
        }
    }

    /**
     * Log current statistics
     */
    private void logStatistics() {
        long total = totalRequests.get();
        long success = successfulRequests.get();
        long failed = failedRequests.get();
        double successRate = total > 0 ? (success * 100.0 / total) : 0.0;

        logger.info("[{}] STATS: Total: {} | Success: {} | Failed: {} | Success Rate: {}%",
                getCurrentTime(), total, success, failed, String.format("%.2f", successRate));
    }

    /**
     * Log final statistics on shutdown
     */
    private void logFinalStatistics() {
        logger.info("=".repeat(80));
        logger.info("FINAL STATISTICS");
        logger.info("=".repeat(80));
        logStatistics();
        logger.info("=".repeat(80));
    }

    /**
     * Log startup banner
     */
    private void logBanner() {
        logger.info("=".repeat(80));
        logger.info("Aurora Blue-Green Deployment Workload Simulator");
        logger.info("Version: 1.0.0");
        logger.info("=".repeat(80));
    }

    /**
     * Log configuration
     */
    private void logConfiguration() {
        logger.info("Configuration:");
        logger.info("  Aurora Endpoint: {}", auroraEndpoint);
        logger.info("  Database Name: {}", databaseName);
        logger.info("  Write Workers: {}", writeWorkers);
        logger.info("  Write Rate: {} writes/sec/worker", writeRate);
        logger.info("  Connection Pool Size: {}", connectionPoolSize);
        logger.info("  Log Interval: {} seconds", logInterval);
        logger.info("  Metrics Enabled: {}", enableMetrics);
        logger.info("=".repeat(80));
    }

    /**
     * Get current time as formatted string
     */
    private static String getCurrentTime() {
        return LocalDateTime.now().format(timeFormatter);
    }

    /**
     * Main entry point
     */
    public static void main(String[] args) {
        // Install JUL to SLF4J bridge to route AWS JDBC Wrapper logging through Log4j2
        // This must be done before any JUL loggers are created
        org.slf4j.bridge.SLF4JBridgeHandler.removeHandlersForRootLogger();
        org.slf4j.bridge.SLF4JBridgeHandler.install();

        // Set JUL root logger level to ALL to allow SLF4J bridge to control filtering
        java.util.logging.Logger.getLogger("").setLevel(java.util.logging.Level.ALL);

        Options options = new Options();

        options.addOption(Option.builder()
                .longOpt("aurora-endpoint")
                .hasArg()
                .required()
                .desc("Aurora cluster writer endpoint (required)")
                .build());

        options.addOption(Option.builder()
                .longOpt("database-name")
                .hasArg()
                .desc("Database name (default: lab_db)")
                .build());

        options.addOption(Option.builder()
                .longOpt("username")
                .hasArg()
                .desc("Database username (default: admin)")
                .build());

        options.addOption(Option.builder()
                .longOpt("password")
                .hasArg()
                .desc("Database password (default: from environment variable DB_PASSWORD)")
                .build());

        options.addOption(Option.builder()
                .longOpt("write-workers")
                .hasArg()
                .type(Number.class)
                .desc("Number of concurrent write workers (default: 10)")
                .build());

        options.addOption(Option.builder()
                .longOpt("write-rate")
                .hasArg()
                .type(Number.class)
                .desc("Writes per second per worker (default: 100)")
                .build());

        options.addOption(Option.builder()
                .longOpt("connection-pool-size")
                .hasArg()
                .type(Number.class)
                .desc("Database connection pool size (default: 100)")
                .build());

        options.addOption(Option.builder()
                .longOpt("log-interval")
                .hasArg()
                .type(Number.class)
                .desc("Statistics log interval in seconds (default: 10)")
                .build());

        options.addOption(Option.builder()
                .longOpt("enable-metrics")
                .desc("Enable Prometheus metrics server on port 8080 (default: false)")
                .build());

        options.addOption("h", "help", false, "Show help message");

        CommandLineParser parser = new DefaultParser();
        HelpFormatter formatter = new HelpFormatter();

        try {
            CommandLine cmd = parser.parse(options, args);

            if (cmd.hasOption("help")) {
                formatter.printHelp("workload-simulator", options);
                System.exit(0);
            }

            String auroraEndpoint = cmd.getOptionValue("aurora-endpoint");
            String databaseName = cmd.getOptionValue("database-name", "lab_db");
            String username = cmd.getOptionValue("username", "admin");
            String password = cmd.getOptionValue("password", System.getenv("DB_PASSWORD"));

            if (password == null || password.isEmpty()) {
                logger.error("Database password not provided. Use --password or set DB_PASSWORD environment variable.");
                System.exit(1);
            }

            int writeWorkers = cmd.hasOption("write-workers")
                    ? ((Number) cmd.getParsedOptionValue("write-workers")).intValue()
                    : 10;
            int writeRate = cmd.hasOption("write-rate")
                    ? ((Number) cmd.getParsedOptionValue("write-rate")).intValue()
                    : 100;
            int connectionPoolSize = cmd.hasOption("connection-pool-size")
                    ? ((Number) cmd.getParsedOptionValue("connection-pool-size")).intValue()
                    : 100;
            int logInterval = cmd.hasOption("log-interval")
                    ? ((Number) cmd.getParsedOptionValue("log-interval")).intValue()
                    : 10;
            boolean enableMetrics = cmd.hasOption("enable-metrics");

            // Validate parameters
            if (writeWorkers < 1) {
                logger.error("Minimum 1 write worker required. Provided: {}", writeWorkers);
                System.exit(1);
            }

            if (connectionPoolSize < writeWorkers) {
                logger.warn("Connection pool size ({}) is less than worker count ({}). " +
                        "This may cause connection contention.", connectionPoolSize, writeWorkers);
            }

            WorkloadSimulator simulator = new WorkloadSimulator(
                    auroraEndpoint, databaseName, username, password,
                    writeWorkers, writeRate, connectionPoolSize, logInterval, enableMetrics
            );

            simulator.start();

        } catch (ParseException e) {
            logger.error("Failed to parse command line arguments: {}", e.getMessage());
            formatter.printHelp("workload-simulator", options);
            System.exit(1);
        } catch (Exception e) {
            logger.error("Failed to start workload simulator", e);
            System.exit(1);
        }
    }
}
