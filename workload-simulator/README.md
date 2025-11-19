# Aurora Workload Simulator

A Java-based workload simulator designed to test Aurora Blue-Green deployment behavior with realistic write operations.

## Features

- **AWS Advanced JDBC Wrapper**: Automatic failover detection and handling
- **Configurable Workload**: Adjustable worker count, write rate, and connection pool
- **Real-time Monitoring**: Console output with success/failure indicators
- **Prometheus Metrics**: Optional metrics export for Kubernetes deployments
- **HikariCP Connection Pool**: High-performance connection pooling
- **Log4j2 Logging**: High-performance logging with automatic file rotation
- **Production-Ready**: Designed for both EC2 and Kubernetes deployments

## Prerequisites

- Java 17 (Amazon Corretto recommended)
- Maven 3.9+
- Access to Aurora MySQL cluster
- Database initialized with tables (see `scripts/init-schema.sh`)

## Building

### Build JAR file

```bash
mvn clean package
```

The fat JAR will be created at `target/workload-simulator-1.0.0.jar`

### Build Docker image

```bash
docker build -t workload-simulator:latest .
```

## Running

### Option 1: EC2 Deployment (Recommended for Beginners)

```bash
# Basic configuration
java -jar workload-simulator.jar \
  --aurora-endpoint my-cluster.cluster-xxxxx.us-east-1.rds.amazonaws.com \
  --database-name lab_db \
  --username admin \
  --password MySecurePassword123 \
  --write-workers 10 \
  --write-rate 100 \
  --connection-pool-size 100

# High load configuration
java -jar workload-simulator.jar \
  --aurora-endpoint my-cluster.cluster-xxxxx.us-east-1.rds.amazonaws.com \
  --database-name lab_db \
  --username admin \
  --password MySecurePassword123 \
  --write-workers 50 \
  --write-rate 200 \
  --connection-pool-size 500
```

### Option 2: Kubernetes Deployment (Advanced)

```bash
# Update secret with your Aurora endpoint and password
kubectl apply -f kubernetes/secret.yaml

# Deploy all resources
kubectl apply -f kubernetes/

# Or use Kustomize
kubectl apply -k kubernetes/

# View logs
kubectl logs -f deployment/workload-simulator

# Scale deployment
kubectl scale deployment workload-simulator --replicas=5
```

## Command-Line Options

| Option | Required | Default | Description |
|--------|----------|---------|-------------|
| `--aurora-endpoint` | Yes | - | Aurora cluster writer endpoint |
| `--database-name` | No | `lab_db` | Database name |
| `--username` | No | `admin` | Database username |
| `--password` | No | `$DB_PASSWORD` | Database password (or set DB_PASSWORD env var) |
| `--write-workers` | No | `10` | Number of concurrent write workers (min: 1) |
| `--write-rate` | No | `100` | Writes per second per worker |
| `--connection-pool-size` | No | `100` | HikariCP connection pool size |
| `--log-interval` | No | `10` | Statistics logging interval in seconds |
| `--enable-metrics` | No | `false` | Enable Prometheus metrics server on port 8080 |

## Output Format

### Console Output

```
================================================================================
Aurora Blue-Green Deployment Workload Simulator
Version: 1.0.0
================================================================================
Configuration:
  Aurora Endpoint: my-cluster.cluster-xxxxx.us-east-1.rds.amazonaws.com
  Database Name: lab_db
  Write Workers: 10
  Write Rate: 100 writes/sec/worker
  Connection Pool Size: 100
  Log Interval: 10 seconds
  Metrics Enabled: false
================================================================================

[2025-01-18 10:15:24.123] SUCCESS: Worker-1 | Host: ip-10-0-1-45 (writer) | Table: test_0001 | INSERT completed | Latency: 12ms
[2025-01-18 10:15:24.234] SUCCESS: Worker-2 | Host: ip-10-0-1-45 (writer) | Table: test_0042 | INSERT completed | Latency: 15ms
[2025-01-18 10:15:34.123] STATS: Total: 1000 | Success: 1000 | Failed: 0 | Success Rate: 100.00%
[2025-01-18 10:16:45.678] ERROR: Worker-5 | Table: test_0123 | connection_lost | Retry 1/5 in 500ms | Error: Communications link failure
[2025-01-18 10:16:46.234] INFO: Worker-5 | Switched to new host: ip-10-0-2-78 (writer) (from: ip-10-0-1-45 (writer))
[2025-01-18 10:16:46.345] SUCCESS: Worker-5 | Host: ip-10-0-2-78 (writer) | Table: test_0123 | INSERT completed | Latency: 234ms (retry 1)
```

### Understanding the Host Field

Each successful write operation logs the Aurora instance hostname and role that handled the request:

- **Format**: `Host: ip-10-0-1-45 (writer)` or `Host: ip-10-0-2-78 (reader)`
- **Purpose**: Track which Aurora instance is serving writes before, during, and after Blue-Green switchover
- **Role Detection**: Automatically detects if the instance is a writer or reader using `@@read_only` variable
  - `(writer)` - Read-write instance that can accept writes
  - `(reader)` - Read-only instance (if you see writes going to a reader, this indicates a configuration error)

**Key Observations During Switchover**:
  - **Before switchover**: All writes go to Blue cluster writer (e.g., `ip-10-0-1-45 (writer)`)
  - **During switchover**: Connection errors occur as Blue writer becomes unavailable
  - **After switchover**: Writes resume on Green cluster writer (e.g., `ip-10-0-2-78 (writer)`)
  - **Host switch notification**: Automatic INFO log when worker connects to a different host

**Example Blue-Green Switchover Sequence**:
```
[10:15:23] SUCCESS: Worker-1 | Host: ip-10-0-1-45 (writer) | ...  ← Blue cluster writer
[10:15:24] SUCCESS: Worker-2 | Host: ip-10-0-1-45 (writer) | ...  ← Blue cluster writer
[10:16:45] ERROR: Worker-5 | connection_lost | Retry 1/5...  ← Switchover begins
[10:16:46] INFO: Worker-5 | Switched to new host: ip-10-0-2-78 (writer) (from: ip-10-0-1-45 (writer))  ← Host switch detected!
[10:16:46] SUCCESS: Worker-5 | Host: ip-10-0-2-78 (writer) | ... (retry 1)  ← Green cluster writer
[10:16:47] INFO: Worker-1 | Switched to new host: ip-10-0-2-78 (writer) (from: ip-10-0-1-45 (writer))
[10:16:47] SUCCESS: Worker-1 | Host: ip-10-0-2-78 (writer) | ...  ← All traffic now on Green
```

**Benefits of Enhanced Host Tracking**:
- **Automatic Host Switch Detection**: INFO-level logs appear whenever a worker switches to a different Aurora instance
- **Role Verification**: Confirms writes are going to writer instances (not readers by mistake)
- **Failover Visibility**: Clear indication of when each worker reconnects to the new cluster
- **Multi-Worker Coordination**: See exactly when each worker completes the switchover

This enhanced visibility helps you confirm that the failover completed successfully and all workload traffic moved to the new cluster writer.

### Enabling Verbose Logging

By default, individual write operations are logged at DEBUG level to avoid excessive console output. To see the host information for **every** write operation:

**Option 1: Set LOG_LEVEL environment variable**
```bash
LOG_LEVEL=DEBUG java -jar workload-simulator.jar \
  --aurora-endpoint <endpoint> \
  --write-workers 1 \
  --log-interval 2
```

**Option 2: Set as Java system property**
```bash
java -DLOG_LEVEL=DEBUG -jar workload-simulator.jar \
  --aurora-endpoint <endpoint> \
  --write-workers 1 \
  --log-interval 2
```

**With DEBUG logging enabled, you'll see:**
```
[2025-01-19 09:15:24.123] SUCCESS: Worker-1 | Host: ip-10-0-1-45 (writer) | Table: test_0001 | INSERT completed | Latency: 12ms
[2025-01-19 09:15:24.134] SUCCESS: Worker-1 | Host: ip-10-0-1-45 (writer) | Table: test_0042 | INSERT completed | Latency: 11ms
[2025-01-19 09:15:24.145] SUCCESS: Worker-1 | Host: ip-10-0-1-45 (writer) | Table: test_0123 | INSERT completed | Latency: 13ms
... (100 log lines per second per worker)
[2025-01-19 09:16:45.678] ERROR: Worker-1 | Table: test_0456 | connection_lost | Retry 1/5 in 500ms
[2025-01-19 09:16:46.123] INFO: Worker-1 | Switched to new host: ip-10-0-2-78 (writer) (from: ip-10-0-1-45 (writer))
[2025-01-19 09:16:46.234] SUCCESS: Worker-1 | Host: ip-10-0-2-78 (writer) | Table: test_0456 | INSERT completed | Latency: 234ms (retry 1)
```

**Note**: At 100 writes/sec per worker, DEBUG logging generates **100 log lines per second per worker**. This is useful for:
- Observing the exact moment of Blue-Green switchover
- Seeing which host handles each write operation in real-time
- Debugging connection issues with detailed timestamps

**For normal testing**, the STATS logs (every 2-10 seconds) provide sufficient visibility without flooding the console. The detailed logs are always written to `workload-simulator.log` regardless of console log level.

### Log File Management

The workload simulator uses **Log4j2** with automatic log rotation to manage disk space:

- **Active log file**: `workload-simulator.log` (current log entries)
- **Rotation trigger**: When log file reaches **10MB**
- **Rotated files**: `workload-simulator-YYYY-MM-DD-N.log` (e.g., `workload-simulator-2025-11-19-1.log`)
- **Retention policy**: Keeps last **7 days** of logs
- **Total size cap**: Maximum **100MB** total across all log files
- **Automatic cleanup**: Old logs are deleted automatically after 7 days

**Benefits**:
- No manual log cleanup required
- Prevents disk space exhaustion during long-running tests
- Preserves recent logs for troubleshooting
- Each rotated file is manageable in size (≤10MB)

## Prometheus Metrics

When `--enable-metrics` is enabled, the following metrics are exposed on port 8080:

- `aurora_write_requests_total{status="success|failure"}` - Total write requests by status
- `aurora_write_latency_seconds` - Write operation latency histogram
- `aurora_connection_errors_total{error_type="..."}` - Connection errors by type
- Standard JVM metrics (heap, threads, GC, etc.)

Access metrics at: `http://localhost:8080/metrics`

## Connection Pool Sizing

**Recommendation**: 10 connections per worker for optimal throughput.

Examples:
- 10 workers → pool size 100
- 50 workers → pool size 500

Minimum pool size should be at least equal to the number of workers to avoid connection contention.

## AWS JDBC Wrapper Configuration

The simulator uses AWS Advanced JDBC Wrapper with the following plugins enabled:

- **Blue-Green Plugin**: Proactively monitors Blue-Green deployment status for optimal switchover
- **Failover Plugin**: Automatic failover detection and handling for general cluster failures
- **Enhanced Failure Monitoring (EFM)**: Proactive connection health monitoring

### Plugin Configuration

**Blue-Green Plugin** (Optimized for Blue-Green Deployments):
- Blue-Green Deployment ID: **1** (required parameter)
- Connection timeout: **30 seconds** (max wait for connections during switchover)
- Switchover timeout: **180 seconds** (3 minutes max for complete switchover)
- **Proactive monitoring**: Detects Blue-Green deployment status changes before switchover
- **Coordinated failover**: Manages connection suspension and redirection across all workers

**Failover Plugin** (General Failover Support):
- Failover timeout: **10 seconds** (aggressive fail-fast for minimal downtime)
- Topology refresh rate: **1 second** (faster detection of new writer)
- Cluster-aware failover: Enabled
- Retry logic: **5 attempts** with exponential backoff (500ms, 1s, 2s, 4s, 8s)

### Why These Aggressive Settings?

The combination of Blue-Green plugin with optimized failover settings provides the best possible switchover experience:

**Blue-Green Plugin Benefits**:
1. **Proactive Monitoring**:
   - Monitors RDS Blue-Green deployment API status continuously
   - Detects `SWITCHING_OVER` status **before** connections fail
   - Prepares connections for imminent switchover (typically 30-60 seconds in advance)

2. **Coordinated Switchover**:
   - All connections across all workers coordinate the switchover
   - Suspends new connections to Blue cluster at the right moment
   - Redirects existing connections to Green cluster seamlessly
   - Minimizes the window where connections are in "unknown" state

3. **Faster Recovery**:
   - No reliance on DNS propagation delays
   - Direct detection of Green cluster readiness
   - Immediate connection establishment to new writer

**Failover Plugin Optimization**:
1. **10-Second Timeout**:
   - Fails fast if detection misses the Blue-Green signal
   - Quick retry instead of blocking for 60 seconds
   - Complements Blue-Green plugin as a fallback mechanism

2. **1-Second Topology Refresh**:
   - Rapid detection of cluster topology changes
   - Works in tandem with Blue-Green plugin for redundancy
   - Ensures no missed switchover events

3. **5 Retry Attempts with Exponential Backoff**:
   - Higher success rate during edge cases
   - Shorter initial delay (500ms) for faster recovery
   - Prevents overwhelming the new writer

### Expected Downtime During Switchover

**With Blue-Green Plugin (Optimal Path)**:
```
Time -60s: Blue-Green plugin detects deployment status = "SWITCHING_OVER"
Time -30s: Plugin prepares for switchover, logs status change
Time -5s:  Plugin begins connection redirection preparation
Time 0s:   Blue-Green switchover triggered by AWS
Time 1-3s: Coordinated connection switchover to Green cluster
Time 3s:   All workers successfully connected to Green writer
```
**Total downtime: ~3-5 seconds** ✅ **(Best Case with Blue-Green Plugin)**

**Without Blue-Green Plugin Detection (Failover Fallback)**:
```
Time 0s:   Switchover triggered - old writer stops accepting writes
Time 0-10s: Connections to Blue fail, failover plugin activates
Time 10s:  Attempt 1 fails (10s timeout) → Wait 500ms
Time 11s:  Attempt 2 succeeds (new topology detected)
```
**Total downtime: ~11 seconds** (reactive failover mode)

**Worst case scenario** (Blue-Green plugin misses signal + all failover attempts delayed):
```
Time 0s:   Switchover triggered
Time 10s:  Attempt 1 fails → Wait 500ms
Time 11s:  Attempt 2 fails → Wait 1s
Time 22s:  Attempt 3 fails → Wait 2s
Time 34s:  Attempt 4 SUCCEEDS
```
**Total downtime: ~34 seconds** (extremely rare, requires plugin communication failure)

### Blue-Green vs. Failover-Only Comparison

| Scenario | Blue-Green Plugin | Failover Only |
|----------|-------------------|---------------|
| **Detection Method** | Proactive API monitoring | Reactive connection failure |
| **Advance Warning** | 30-60 seconds before switchover | None (detects after failure) |
| **Coordination** | Cross-connection coordination | Individual connection handling |
| **Typical Downtime** | **3-5 seconds** | 11-20 seconds |
| **Best Case** | **3 seconds** | 5 seconds |
| **Worst Case** | 34 seconds | 66 seconds |

The Blue-Green plugin provides **60-75% reduction in downtime** compared to failover-only configurations.

## Testing Blue-Green Deployment

1. **Start the workload simulator** with desired configuration
2. **Verify workload is running** - Watch console output for successful writes
3. **Initiate Blue-Green deployment** via AWS Console or CLI
4. **Keep simulator running** - Do NOT stop during upgrade
5. **Observe console output** during switchover:
   - Look for connection errors
   - Note timestamp of failures
   - Observe failover behavior
6. **Complete switchover** when ready
7. **Monitor logs** during switchover for:
   - Connection interruptions
   - Automatic reconnection
   - Time to recovery
8. **Validate post-upgrade**:
   - Verify workload continues successfully
   - Review total failures vs. successes

## Troubleshooting

### Connection Pool Exhausted

```
ERROR: HikariPool - Connection is not available
```

**Solution**: Increase `--connection-pool-size` or reduce `--write-workers`

### High Latency

```
WARNING: Write latency exceeds 1000ms
```

**Solution**:
- Reduce `--write-rate`
- Check Aurora instance size
- Verify network connectivity

### Continuous Connection Errors

```
ERROR: Worker-X | connection_lost | Error: Communications link failure
```

**Solution**:
- Verify Aurora endpoint is correct
- Check security group allows access
- Verify Aurora cluster is running

## Architecture

```
┌─────────────────────────────────────────┐
│      Workload Simulator                 │
│                                         │
│  ┌─────────────────────────────────┐   │
│  │   HikariCP Connection Pool      │   │
│  │   (100+ connections)            │   │
│  └──────────┬──────────────────────┘   │
│             │                           │
│  ┌──────────▼──────────┐               │
│  │  AWS JDBC Wrapper   │               │
│  │  - Failover Plugin  │               │
│  │  - EFM Plugin       │               │
│  └──────────┬──────────┘               │
│             │                           │
│  ┌──────────▼──────────────────────┐   │
│  │  Write Workers (10+)            │   │
│  │  - Concurrent INSERT operations │   │
│  │  - Random table selection       │   │
│  │  - Rate limiting                │   │
│  └─────────────────────────────────┘   │
└─────────────────────────────────────────┘
              │
              │ JDBC Connection
              ▼
┌─────────────────────────────────────────┐
│     Aurora MySQL Cluster                │
│                                         │
│  ┌─────────────┐    ┌─────────────┐   │
│  │   Writer    │    │   Reader    │   │
│  │  Instance   │◄───│  Instance   │   │
│  └─────────────┘    └─────────────┘   │
└─────────────────────────────────────────┘
```

## How It Works

### Overview

The Aurora Blue-Green Deployment Lab uses two components that work together:

1. **init-schema.sh** - Prepares the database with production-scale metadata
2. **workload-simulator** - Generates sustained write workload to observe switchover behavior

### init-schema.sh: Database Preparation

#### Purpose
Creates **12,000 tables** in the Aurora database to simulate a production environment with heavy metadata overhead. This is critical because Blue-Green deployments need to clone metadata, and the more tables you have, the longer the switchover may take.

#### Schema Structure

Each of the 12,000 tables follows this pattern:

```sql
CREATE TABLE test_0001 (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    col1 VARCHAR(255) NOT NULL,
    col2 INT DEFAULT 0,
    col3 TEXT,
    col4 DECIMAL(10,2) DEFAULT 0.00,
    col5 BIGINT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_col1 (col1),
    INDEX idx_col2 (col2),
    INDEX idx_col5 (col5)
) ENGINE=InnoDB;
```

- **Table names**: `test_0001`, `test_0002`, ... `test_12000`
- **5 data columns**: col1 (VARCHAR), col2 (INT), col3 (TEXT), col4 (DECIMAL), col5 (BIGINT)
- **3 indexes**: Primary key + 2 secondary indexes
- **Timestamps**: Auto-tracked creation and update times

#### Execution Flow

```
1. Validate Parameters
   ├─ Check Aurora endpoint
   ├─ Check credentials
   └─ Test connection

2. Create Database
   └─ CREATE DATABASE IF NOT EXISTS lab_db

3. Create Tables in Batches
   ├─ Batch 1: tables 1-100 (parallel)
   ├─ Batch 2: tables 101-200 (parallel)
   ├─ ...
   └─ Batch 120: tables 11901-12000

4. Parallel Execution
   ├─ 4 parallel connections by default
   ├─ Each creates 100 tables per batch
   └─ Speeds up from ~2 hours to ~30-60 minutes

5. Optional: Insert Initial Data
   └─ 1 row per table (minimal data)

6. Verify Table Count
   └─ Query information_schema to confirm 12,000 tables exist
```

#### Why 12,000 Tables?

This simulates a **production-scale database** where:
- Large enterprises often have thousands of tables
- More tables = more metadata to clone during Blue-Green switchover
- Tests Aurora's ability to handle heavy metadata operations
- Reveals any delays or issues during the switchover phase

### Workload Simulator: Continuous Write Operations

#### Purpose
Generates **continuous write operations** to the Aurora cluster while you perform the Blue-Green deployment, allowing you to observe:
- Whether writes are interrupted during switchover
- How long the interruption lasts
- How the AWS JDBC Wrapper handles automatic failover
- Connection recovery behavior

#### Write Worker Loop

Each of the 10+ workers runs this loop continuously:

```
LOOP (until stopped):
  1. Get connection from pool
  2. Select random table: test_XXXX (where XXXX = 0001 to 12000)
  3. Generate random data
  4. Execute INSERT:
     INSERT INTO test_XXXX (col1, col2, col3, col4, col5)
     VALUES (random_string, random_int, random_text, random_decimal, timestamp)
  5. Log result:
     - SUCCESS: [timestamp] Worker-5 | Table: test_0042 | INSERT completed | Latency: 12ms
     - ERROR: [timestamp] Worker-5 | Connection lost | Error: Communications link failure
  6. If error: AWS JDBC Wrapper attempts reconnection
  7. Sleep for rate limiting (e.g., 10ms for 100 TPS)
END LOOP
```

#### Key Features

**1. AWS Advanced JDBC Wrapper Integration**

This is the critical component that makes the simulator valuable:

```java
jdbc:aws-wrapper:mysql://cluster-endpoint:3306/lab_db
```

The wrapper provides:
- **Automatic Failover Detection**: Detects when the Blue-Green switchover happens
- **Cluster Topology Awareness**: Knows about reader and writer instances
- **Connection State Tracking**: Identifies broken connections immediately
- **Automatic Reconnection**: Attempts to reconnect to the new writer endpoint

**2. Write Operation Pattern**

Each worker continuously:
- Selects a **random table** from the 12,000 tables (test_0001 to test_12000)
- Inserts a **new row** with random data in all 5 columns
- Tracks **latency** for each operation
- Logs **success or failure** with timestamps

This randomness ensures:
- Even distribution across all 12,000 tables
- Realistic workload pattern
- Easy to spot interruptions during switchover

**3. Real-Time Visibility During Switchover**

```
[2025-01-18 10:15:24.123] SUCCESS: Worker-1 | Table: test_0001 | INSERT completed | Latency: 12ms
[2025-01-18 10:15:24.234] SUCCESS: Worker-2 | Table: test_0042 | INSERT completed | Latency: 15ms
...
DURING BLUE-GREEN SWITCHOVER:
[2025-01-18 10:16:45.678] ERROR: Worker-5 | Connection lost | Error: Communications link failure
[2025-01-18 10:16:45.679] ERROR: Worker-2 | Connection lost | Error: Communications link failure
[2025-01-18 10:16:45.789] INFO: Worker-5 | Attempting reconnection...
[2025-01-18 10:16:46.123] SUCCESS: Worker-5 | Reconnected successfully
[2025-01-18 10:16:46.234] SUCCESS: Worker-2 | Reconnected successfully
```

This output lets you:
- **See exactly when** the switchover happened (timestamp of first failures)
- **Count failed transactions** during the switchover window
- **Measure recovery time** (time between first failure and successful reconnection)
- **Verify zero-downtime** (or identify the actual downtime duration)

### Complete Workflow

```
Phase 1: PREPARATION
├─ Run init-schema.sh
│  └─ Creates 12,000 tables in Aurora (takes 30-60 min)
└─ Result: Database with production-scale metadata

Phase 2: BASELINE WORKLOAD
├─ Start workload-simulator
│  ├─ 10 workers × 100 writes/sec = 1,000 writes/sec total
│  └─ Observe all SUCCESS messages
└─ Result: Stable write workload established

Phase 3: BLUE-GREEN DEPLOYMENT
├─ Initiate Blue-Green deployment (AWS Console or CLI)
│  ├─ Aurora clones the cluster (Blue → Green)
│  ├─ Applies engine upgrade to Green cluster
│  └─ All 12,000 tables are copied to Green
└─ Wait for Green cluster to be ready (10-60 minutes depending on data size)

Phase 4: SWITCHOVER (THE CRITICAL MOMENT)
├─ Trigger switchover (AWS CLI)
├─ Aurora redirects writer endpoint: Blue → Green
├─ Workload simulator experiences:
│  ├─ Connection errors (2-5 seconds typically)
│  ├─ AWS JDBC Wrapper detects topology change
│  ├─ Automatic reconnection to new writer
│  └─ Writes resume successfully
└─ Result: You observe the exact behavior and downtime (if any)

Phase 5: VALIDATION
├─ Workload continues successfully on Green cluster
├─ Review console logs to count failed transactions
├─ Verify Aurora version upgrade: SELECT @@aurora_version;
└─ Calculate actual downtime and success rate
```

### Why This Design is Effective

1. **12,000 Tables = Realistic Test**
   - Tests Aurora's ability to handle metadata-heavy workloads
   - Longer table clone time = longer switchover preparation
   - Reveals if metadata size impacts downtime

2. **Continuous Writes = Visibility**
   - You can't miss the switchover moment
   - Timestamps show exact failure and recovery times
   - Easy to calculate downtime (if any)

3. **AWS JDBC Wrapper = Production-Ready**
   - This is how real applications should connect to Aurora
   - Demonstrates best practices for failover handling
   - Shows automatic recovery without manual intervention

4. **Random Table Selection = Even Distribution**
   - Tests all 12,000 tables, not just a few
   - Avoids hotspots
   - Realistic production pattern

### Example Test Scenario

**Setup:**
- 12,000 tables created (init-schema.sh)
- Workload running: 10 workers @ 100 writes/sec = 1,000 TPS total
- Aurora version: 3.04 → Upgrading to 3.10

**During Blue-Green Switchover:**
```
[10:16:45.678] SUCCESS: Worker-1 | Table: test_5432 | INSERT completed | Latency: 12ms
[10:16:45.789] SUCCESS: Worker-2 | Table: test_0987 | INSERT completed | Latency: 14ms
[10:16:45.890] ERROR: Worker-3 | Connection lost | Communications link failure  ← SWITCHOVER STARTS
[10:16:45.901] ERROR: Worker-1 | Connection lost | Communications link failure
[10:16:45.912] ERROR: Worker-2 | Connection lost | Communications link failure
[10:16:46.000] INFO: Worker-3 | Attempting reconnection...
[10:16:47.234] SUCCESS: Worker-3 | Reconnected successfully  ← RECOVERY (1.3 seconds downtime)
[10:16:47.345] SUCCESS: Worker-1 | Reconnected successfully
[10:16:47.456] SUCCESS: Worker-2 | Reconnected successfully
[10:16:47.567] SUCCESS: Worker-4 | Table: test_8765 | INSERT completed | Latency: 15ms
```

**Analysis:**
- **Downtime**: ~1.3 seconds
- **Failed transactions**: ~13 writes (1,000 TPS × 1.3 sec)
- **Recovery**: Automatic, no manual intervention
- **Success rate**: 99.87% overall

This data helps you understand Aurora Blue-Green deployment behavior in your specific environment!

## License

See [LICENSE](../LICENSE) file for details.
