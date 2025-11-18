# Aurora Blue-Green Deployment Lab

## Project Overview

This project demonstrates the Blue-Green upgrade feature of Amazon Aurora. It provides a complete lab environment to practice the Blue-Green upgrade process with real-world workload simulation and monitoring capabilities.

## Purpose

The lab environment enables hands-on practice with Aurora Blue-Green deployments, allowing users to:
- Experience zero-downtime upgrades in a controlled environment
- Monitor the impact of Blue-Green switchovers on application workloads
- Understand the behavior of write operations during the upgrade process

## Lab Components

### 2.1 Aurora Database Infrastructure
- **Infrastructure as Code**: Pulumi (Golang)
- **Database**: Amazon Aurora MySQL-Compatible
  - Initial Version: 3.04
  - Target Version: 3.10
  - Database Schema: 12,000 tables to simulate production-scale metadata overhead and test Blue-Green deployment performance with large schemas
- **Cluster Configuration**:
  - 1 Writer instance (r6g.xlarge) - Primary endpoint for write operations
  - 1 Reader instance (r6g.xlarge) - Standby for high availability
  - Multi-AZ deployment for production-like reliability testing

### 2.2 Workload Simulator
- **Language**: Java
- **JDK Version**: Amazon Corretto 17
- **JDBC Driver**: AWS Advanced JDBC Wrapper 2.6.6
- **Workload Design**: Write-only workload targeting the writer endpoint to simulate production write operations during Blue-Green switchover
- **Deployment Options**:
  - **Option 1 (Recommended for Beginners)**: EC2 instance with manual execution
    - Simple command-line execution with direct console output
    - Real-time log output showing success/failed connections
    - Easy to start, stop, and observe
    - Ideal for learning and understanding the Blue-Green switchover behavior
  - **Option 2 (Advanced)**: Kubernetes (EKS) deployment
    - Containerized deployment for scaled testing
    - Supports multiple pod instances for high-concurrency scenarios
    - Integrated with Prometheus/Grafana for advanced monitoring
    - Suitable for stress testing and production-like simulations
- **Worker Configuration**:
  - Minimum 10 write workers (configurable via parameters)
  - Workers can be scaled dynamically based on testing requirements
- **Functionality**:
  - Simulates realistic write workloads against the Aurora cluster
  - Uses AWS Advanced JDBC Wrapper for enhanced Aurora connectivity features:
    - Automatic failover detection and handling during Blue-Green switchover
    - Connection state tracking to detect interrupted transactions
    - Enhanced monitoring and metrics for connection health
  - Configurable workload parameters:
    - Number of write workers
    - Write request rate per worker
    - Connection pool settings
- **Output & Monitoring**:
  - Real-time console log output with timestamps
  - Success/failure indicators for each database operation
  - Connection error details during switchover events
  - Summary statistics (total requests, success rate, error count)
  - Optional metrics export to Prometheus (EKS deployment only)

### 2.3 Kubernetes Infrastructure (Optional - Advanced Testing)
- **Infrastructure as Code**: Pulumi (Golang)
- **Container Orchestration**: Amazon EKS (Elastic Kubernetes Service)
- **Use Case**: Advanced testing scenarios requiring high concurrency and distributed workloads
- **Deployment Architecture**:
  - Designed to support multiple workload simulator instances
  - Resource allocation per pod: 2 vCPU, 4GB RAM (supports 10-50 write workers)
  - Horizontal Pod Autoscaling (HPA) support for dynamic scaling based on CPU/memory utilization
  - Node group sizing optimized for high-concurrency workload scenarios (50+ total workers across pods)
- **Purpose**:
  - Hosts and manages containerized workload simulators for scaled testing
  - Enables easy horizontal scaling of workload generators
  - Provides infrastructure for the monitoring stack (Prometheus/Grafana integration)
- **Note**: EC2-based execution is recommended for initial learning; EKS deployment is optional for advanced use cases

### 2.4 Real-time Monitoring
- **Monitoring Stack**:
  - Amazon Managed Service for Prometheus (AMP) for metrics collection and storage
  - Amazon Managed Grafana (AMG) for real-time dashboard visualization
  - AWS Distro for OpenTelemetry (ADOT) Collector for metrics ingestion
  - Custom application metrics exported via Prometheus client library
- **Architecture Benefits**:
  - Fully managed, highly available monitoring infrastructure
  - Automatic scaling and no operational overhead
  - Native AWS IAM integration for secure access
  - Cross-AZ redundancy for reliability
- **Metrics Tracked**:
  - Total write requests sent
  - Successful write operations
  - Failed write operations
  - Response time percentiles (p50, p95, p99)
  - Connection status and errors
  - JDBC wrapper failover events
- **Monitoring Capabilities**:
  - Real-time dashboard visualization with 1-second refresh rate
  - Tracks workload performance during Blue-Green switchover
  - Identifies any service disruptions or degradation
  - Long-term historical data retention for post-upgrade analysis

## Technology Stack

| Component | Technology |
|-----------|-----------|
| Infrastructure as Code | Pulumi (Golang) |
| Database | Amazon Aurora MySQL 3.04 → 3.10 |
| Application Language | Java (Amazon Corretto 17) |
| JDBC Driver | AWS Advanced JDBC Wrapper 2.6.6 |
| Workload Simulator Host | EC2 (Amazon Linux 2023) - Primary<br>Amazon EKS - Optional for advanced testing |
| Monitoring | Amazon Managed Service for Prometheus (AMP) + Amazon Managed Grafana (AMG) |
| Metrics Collection | AWS Distro for OpenTelemetry (ADOT) |

## Getting Started

### Prerequisites
- AWS Account with appropriate permissions
- Pulumi CLI installed
- Go 1.21+ installed
- Docker installed
- kubectl installed
- AWS CLI configured

### Deployment Steps

#### Option 1: EC2-Based Deployment (Recommended for Beginners)

1. **Deploy Aurora Cluster**
   ```bash
   cd infrastructure/aurora
   pulumi up
   ```

2. **Initialize Database Schema**
   ```bash
   # Run the schema initialization script to create 12,000 tables
   cd scripts
   ./init-schema.sh

   # This process may take 30-60 minutes to complete
   # Progress will be logged to schema-init.log
   ```

   The schema initialization creates:
   - 12,000 tables with identical structure
   - Each table has a primary key and 5 data columns
   - Minimal initial data (1 row per table) to establish baseline

3. **Deploy EC2 Instance for Workload Simulator**
   ```bash
   cd infrastructure/ec2
   pulumi up
   ```

   This will provision:
   - EC2 instance (t3.xlarge) with Amazon Linux 2023
   - Amazon Corretto 17 pre-installed
   - Security group allowing outbound access to Aurora
   - Workload simulator JAR pre-deployed

4. **Run Workload Simulator**
   ```bash
   # SSH into the EC2 instance
   ssh -i your-key.pem ec2-user@<ec2-public-ip>

   # Navigate to workload simulator directory
   cd /opt/workload-simulator

   # Start the workload simulator (manual execution)
   java -jar workload-simulator.jar \
     --aurora-endpoint <your-aurora-cluster-endpoint> \
     --write-workers 10 \
     --write-rate 100 \
     --connection-pool-size 100

   # Output will show real-time success/failure logs
   # Press Ctrl+C to stop the simulator
   ```

#### Option 2: EKS-Based Deployment (Advanced - For Scaled Testing)

3. **Deploy EKS Cluster (Optional)**
   ```bash
   cd infrastructure/eks
   pulumi up
   ```

4. **Deploy Monitoring Stack (Optional)**
   ```bash
   # Create Amazon Managed Service for Prometheus workspace
   cd infrastructure/monitoring
   pulumi up

   # Deploy ADOT Collector to EKS for metrics collection
   kubectl apply -f adot-collector-config.yaml
   ```

   This will provision:
   - Amazon Managed Service for Prometheus (AMP) workspace
   - Amazon Managed Grafana (AMG) workspace with IAM authentication
   - ADOT Collector deployment in EKS cluster
   - Pre-configured Grafana dashboard for Blue-Green monitoring

5. **Build and Deploy Workload Simulator to EKS (Optional)**
   ```bash
   cd workload-simulator
   docker build -t workload-simulator:latest .
   kubectl apply -f kubernetes/
   ```

6. **Access Monitoring Dashboard (Optional)**
   ```bash
   # Get the Amazon Managed Grafana workspace URL
   pulumi stack output grafana-workspace-url

   # Access via web browser (SSO or IAM authentication)
   # Dashboard: "Aurora Blue-Green Deployment Monitor"
   ```

   > **Note**: Amazon Managed Grafana uses AWS SSO or IAM Identity Center for authentication. Ensure your AWS account has the necessary permissions configured.

## Blue-Green Upgrade Process

### EC2-Based Testing Flow (Recommended)

1. **Start the workload simulator** on EC2 instance with desired parameters (see step 4 above)
2. **Verify workload is running** - Watch the console output showing successful write operations
3. **Initiate Blue-Green deployment** via AWS Console or CLI:
   ```bash
   aws rds create-blue-green-deployment \
     --blue-green-deployment-name aurora-upgrade-test \
     --source-arn <source-cluster-arn> \
     --target-engine-version 8.0.mysql_aurora.3.10.0
   ```
4. **Keep the workload simulator running** - Do NOT stop the simulator during the upgrade
5. **Observe the console output** during the Blue-Green switchover:
   - Look for connection errors or transaction failures
   - Note the timestamp when errors occur (if any)
   - Observe the JDBC wrapper failover behavior
6. **Complete the switchover** when ready:
   ```bash
   aws rds switchover-blue-green-deployment \
     --blue-green-deployment-identifier <deployment-id>
   ```
7. **Monitor the logs** during switchover - This is the critical moment to observe:
   - Any connection interruptions
   - Automatic reconnection attempts
   - Time to recovery
8. **Validate post-upgrade**:
   - Verify workload continues successfully
   - Check Aurora cluster version: `SELECT @@aurora_version;`
   - Review the log output for total failures vs. successes

### EKS-Based Testing Flow (Optional - Advanced)

1. Deploy workload simulator pods to EKS
2. Monitor metrics in Amazon Managed Grafana
3. Follow steps 3-8 above
4. Use Grafana dashboards to visualize switchover impact

## Workload Simulator Configuration

### Command-Line Parameters

- `--aurora-endpoint`: Aurora cluster writer endpoint (required)
- `--database-name`: Database name (default: lab_db)
- `--write-workers`: Number of concurrent write workers (minimum: 10, default: 10)
- `--write-rate`: Writes per second per worker (default: 100)
- `--connection-pool-size`: Database connection pool size (default: 100)
- `--log-interval`: Statistics log interval in seconds (default: 10)

### EC2 Execution Examples

**Basic Configuration (Recommended for Testing):**
```bash
java -jar workload-simulator.jar \
  --aurora-endpoint my-cluster.cluster-xxxxx.us-east-1.rds.amazonaws.com \
  --database-name lab_db \
  --write-workers 10 \
  --write-rate 100 \
  --connection-pool-size 100
```

**High Load Configuration:**
```bash
java -jar workload-simulator.jar \
  --aurora-endpoint my-cluster.cluster-xxxxx.us-east-1.rds.amazonaws.com \
  --database-name lab_db \
  --write-workers 50 \
  --write-rate 200 \
  --connection-pool-size 500
```

> **Note**: Connection pool sizing recommendation: 10 connections per worker for optimal throughput. Minimum pool size should be at least equal to the number of workers.

### Sample Console Output

```
[2025-01-18 10:15:23.456] INFO: Workload Simulator Started
[2025-01-18 10:15:23.457] INFO: Aurora Endpoint: my-cluster.cluster-xxxxx.us-east-1.rds.amazonaws.com
[2025-01-18 10:15:23.458] INFO: Workers: 10, Rate: 100 writes/sec/worker
[2025-01-18 10:15:24.123] SUCCESS: Worker-1 | Table: test_0001 | INSERT completed | Latency: 12ms
[2025-01-18 10:15:24.234] SUCCESS: Worker-2 | Table: test_0042 | INSERT completed | Latency: 15ms
[2025-01-18 10:15:24.345] SUCCESS: Worker-3 | Table: test_0123 | INSERT completed | Latency: 11ms
...
[2025-01-18 10:15:34.123] STATS: Total: 1000 | Success: 1000 | Failed: 0 | Success Rate: 100.00%
[2025-01-18 10:16:45.678] ERROR: Worker-5 | Connection lost | Error: Communications link failure
[2025-01-18 10:16:45.789] INFO: Worker-5 | Attempting reconnection...
[2025-01-18 10:16:46.123] SUCCESS: Worker-5 | Reconnected successfully
```

### EKS Deployment (Optional - Advanced)

**Kubernetes Deployment Scaling:**
```bash
# Scale the number of workload simulator pods
kubectl scale deployment workload-simulator --replicas=5

# Each pod runs with configurable workers
# Total capacity: 5 pods × 10 write workers = 50 write workers

# View logs from all pods
kubectl logs -l app=workload-simulator -f
```

## Project Structure

```
aurora-blue-green-deployment-lab/
├── infrastructure/
│   ├── aurora/          # Pulumi code for Aurora cluster
│   ├── ec2/             # Pulumi code for EC2 workload simulator host
│   ├── eks/             # (Optional) Pulumi code for EKS cluster
│   └── monitoring/      # (Optional) Pulumi code for AMP and AMG setup
├── scripts/
│   └── init-schema.sh   # Database schema initialization script
├── workload-simulator/  # Java workload simulator application
│   ├── src/             # Java source code
│   ├── pom.xml          # Maven configuration
│   ├── Dockerfile       # (Optional) Container image for EKS deployment
│   └── kubernetes/      # (Optional) K8s manifests for EKS deployment
├── monitoring/          # (Optional) Advanced monitoring for EKS
│   ├── adot-collector-config.yaml  # ADOT Collector configuration
│   └── dashboards/      # Pre-configured Grafana dashboard JSON
└── docs/                # Additional documentation
```

## Learning Objectives

By completing this lab, you will:
- Understand Aurora Blue-Green deployment architecture
- Learn how to monitor database upgrades in real-time
- Experience zero-downtime upgrade procedures
- Gain insights into workload behavior during infrastructure changes
- Practice scaling workloads to test different scenarios

## Resources

- [Aurora Blue-Green Deployments Documentation](https://docs.aws.amazon.com/AmazonRDS/latest/AuroraUserGuide/blue-green-deployments.html)
- [Amazon Aurora MySQL Version 3.10 Release Notes](https://docs.aws.amazon.com/AmazonRDS/latest/AuroraMySQLReleaseNotes/AuroraMySQL.Updates.3010.html)
- [Pulumi AWS Provider Documentation](https://www.pulumi.com/registry/packages/aws/)

## License

See [LICENSE](LICENSE) file for details.
