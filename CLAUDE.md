# Aurora Blue-Green Deployment Lab

## Project Overview

This project demonstrates the Blue-Green upgrade feature of Amazon Aurora. It provides a complete lab environment to practice the Blue-Green upgrade process with real-world workload simulation and monitoring capabilities.

## Purpose

The lab environment enables hands-on practice with Aurora Blue-Green deployments, allowing users to:
- Experience zero-downtime upgrades in a controlled environment
- Monitor the impact of Blue-Green switchovers on application workloads
- Understand the behavior of read and write operations during the upgrade process

## Lab Components

### 2.1 Aurora Database Infrastructure
- **Infrastructure as Code**: Pulumi (Golang)
- **Database**: Amazon Aurora MySQL-Compatible
  - Initial Version: 3.04
  - Target Version: 3.10
  - Database Schema: 12,000 tables for realistic production-scale testing
- **Cluster Configuration**:
  - 1 Writer instance
  - 1 Reader instance
  - Multi-AZ deployment (instances in different Availability Zones)

### 2.2 Workload Simulator
- **Language**: Java
- **JDK Version**: Amazon Corretto 17
- **Container Base Image**: Amazon Linux 2023
- **JDBC Driver**: AWS Advanced JDBC Wrapper 2.6.6
- **Worker Configuration**:
  - Minimum 10 write workers (configurable via parameters)
  - Minimum 60 read workers (configurable via parameters)
  - Workers can be scaled dynamically based on testing requirements
- **Functionality**:
  - Simulates realistic read/write workloads against the Aurora cluster
  - Supports horizontal scaling (scale out/in)
  - Uses AWS Advanced JDBC Wrapper for enhanced Aurora connectivity features
  - Configurable workload parameters:
    - Number of write workers
    - Number of read workers
    - Read/write ratio
    - Request rate
    - Connection pool settings

### 2.3 Kubernetes Infrastructure
- **Infrastructure as Code**: Pulumi (Golang)
- **Container Orchestration**: Amazon EKS (Elastic Kubernetes Service)
- **Deployment Architecture**:
  - Designed to support multiple workload simulator instances
  - Resource allocation configured for 10+ write workers and 60+ read workers per pod
  - Horizontal Pod Autoscaling (HPA) support for dynamic scaling
  - Node group sizing optimized for high-concurrency workload scenarios
- **Purpose**:
  - Hosts and manages the Java workload simulator
  - Enables easy scaling of workload generators
  - Provides infrastructure for the monitoring stack

### 2.4 Real-time Monitoring
- **Metrics Tracked**:
  - Total read requests sent
  - Total write requests sent
  - Successful read operations
  - Successful write operations
  - Failed read operations
  - Failed write operations
  - Response time percentiles (p50, p95, p99)
  - Connection status and errors
- **Monitoring Capabilities**:
  - Real-time dashboard visualization
  - Tracks workload performance during Blue-Green switchover
  - Identifies any service disruptions or degradation

## Technology Stack

| Component | Technology |
|-----------|-----------|
| Infrastructure as Code | Pulumi (Golang) |
| Database | Amazon Aurora MySQL 3.04 → 3.10 |
| Application Language | Java (Amazon Corretto 17) |
| Container Platform | Amazon EKS |
| Base Container Image | Amazon Linux 2023 |

## Getting Started

### Prerequisites
- AWS Account with appropriate permissions
- Pulumi CLI installed
- Go 1.21+ installed
- Docker installed
- kubectl installed
- AWS CLI configured

### Deployment Steps

1. **Deploy Aurora Cluster**
   ```bash
   cd infrastructure/aurora
   pulumi up
   ```

2. **Deploy EKS Cluster**
   ```bash
   cd infrastructure/eks
   pulumi up
   ```

3. **Build and Deploy Workload Simulator**
   ```bash
   cd workload-simulator
   docker build -t workload-simulator:latest .
   kubectl apply -f kubernetes/
   ```

4. **Access Monitoring Dashboard**
   ```bash
   kubectl port-forward svc/monitoring-dashboard 3000:3000
   ```

## Blue-Green Upgrade Process

1. Start the workload simulator with desired parameters
2. Verify monitoring dashboard shows healthy metrics
3. Initiate Blue-Green deployment via AWS Console or CLI
4. Observe workload behavior during the upgrade
5. Monitor for any failed requests or connection errors
6. Complete the switchover when ready
7. Validate upgraded cluster version and application stability

## Workload Simulator Configuration

The workload simulator accepts the following parameters:

- `--write-workers`: Number of concurrent write workers (minimum: 10, default: 10)
- `--read-workers`: Number of concurrent read workers (minimum: 60, default: 60)
- `--write-rate`: Writes per second per worker
- `--read-rate`: Reads per second per worker
- `--duration`: Test duration (default: unlimited)
- `--connection-pool-size`: Database connection pool size

### Example Configurations

**Minimum Configuration:**
```bash
java -jar workload-simulator.jar \
  --write-workers 10 \
  --read-workers 60 \
  --write-rate 100 \
  --read-rate 500 \
  --connection-pool-size 100
```

**High Load Configuration:**
```bash
java -jar workload-simulator.jar \
  --write-workers 50 \
  --read-workers 200 \
  --write-rate 200 \
  --read-rate 1000 \
  --connection-pool-size 300
```

**Kubernetes Deployment Scaling:**
```bash
# Scale the number of workload simulator pods
kubectl scale deployment workload-simulator --replicas=5

# Each pod runs with configurable workers
# Total capacity: 5 pods × (10 write + 60 read workers) = 50 write + 300 read workers
```

## Project Structure

```
aurora-blue-green-deployment-lab/
├── infrastructure/
│   ├── aurora/          # Pulumi code for Aurora cluster
│   └── eks/             # Pulumi code for EKS cluster
├── workload-simulator/  # Java workload simulator
│   ├── src/
│   ├── Dockerfile
│   └── kubernetes/      # K8s manifests for deployment
├── monitoring/          # Monitoring dashboard and configs
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
