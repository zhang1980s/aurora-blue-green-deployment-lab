# Aurora Blue-Green Deployment Lab

A hands-on lab environment for practicing Amazon Aurora MySQL Blue-Green deployments with real-world workload simulation and zero-downtime upgrades.

## Overview

This project provides a complete infrastructure setup to test Aurora's Blue-Green deployment feature, allowing you to:

- Experience zero-downtime database upgrades in a controlled environment
- Monitor the impact of Blue-Green switchovers on application workloads
- Understand connection behavior during infrastructure changes

**What You'll Deploy:**
- Amazon Aurora MySQL cluster (v3.04 → v3.10 upgrade path)
- Java-based workload simulator with 10+ concurrent write workers
- VPC with multi-AZ deployment across public and private subnets
- Optional: Amazon EKS cluster for advanced testing scenarios

## Quick Start

### Prerequisites

- AWS Account with appropriate permissions
- [Pulumi CLI](https://www.pulumi.com/docs/get-started/install/) installed
- [Go 1.21+](https://go.dev/doc/install) installed
- [Java 17](https://docs.aws.amazon.com/corretto/latest/corretto-17-ug/downloads-list.html) (Amazon Corretto recommended)
- [Maven 3.9+](https://maven.apache.org/download.cgi) installed
- [AWS CLI](https://aws.amazon.com/cli/) configured

### Deploy Infrastructure (Automated)

The fastest way to get started:

```bash
# Clone the repository
git clone https://github.com/your-org/aurora-blue-green-deployment-lab.git
cd aurora-blue-green-deployment-lab

# Run automated deployment
cd infrastructure
./deploy.sh
```

The deployment script will:
1. ✅ Check prerequisites
2. 🔐 Prompt for configuration (region, passwords, etc.)
3. 🌐 Deploy VPC and networking (2-3 minutes)
4. 🗄️ Deploy Aurora cluster (10-15 minutes)
5. 💻 Deploy EC2 workload simulator (2-3 minutes)

**Total deployment time: ~15-20 minutes**

### Manual Deployment

If you prefer step-by-step control:

```bash
# 1. Deploy VPC
cd infrastructure/vpc
pulumi up

# 2. Deploy Aurora
cd ../aurora
pulumi config set --secret masterPassword "YourSecurePassword123!"
pulumi up

# 3. Deploy EC2
cd ../ec2
pulumi config set keyName "your-key-pair-name"
pulumi up
```

### Initialize Database Schema

Create 12,000 test tables (takes 30-60 minutes):

```bash
cd scripts
./init-schema.sh \
  --endpoint <aurora-cluster-endpoint> \
  --password <your-password> \
  --tables 12000
```

### Run Workload Simulator

```bash
# SSH into EC2 instance
ssh -i your-key.pem ec2-user@<ec2-public-ip>

# Upload workload simulator JAR
# (From your local machine)
cd workload-simulator
mvn clean package
scp -i your-key.pem target/workload-simulator-1.0.0.jar \
  ec2-user@<ec2-public-ip>:/opt/workload-simulator/

# Run the simulator
cd /opt/workload-simulator
java -jar workload-simulator-1.0.0.jar \
  --aurora-endpoint <cluster-endpoint> \
  --database-name lab_db \
  --username admin \
  --password <your-password> \
  --write-workers 10 \
  --write-rate 100
```

## Testing Blue-Green Deployment

### Step 1: Start Workload

Ensure the workload simulator is running and showing successful writes:

```
[2025-01-19 10:15:24.123] SUCCESS: Worker-1 | Table: test_0001 | INSERT completed | Latency: 12ms
[2025-01-19 10:15:24.234] SUCCESS: Worker-2 | Table: test_0042 | INSERT completed | Latency: 15ms
```

### Step 2: Create Blue-Green Deployment

```bash
aws rds create-blue-green-deployment \
  --blue-green-deployment-name aurora-upgrade-test \
  --source-arn <source-cluster-arn> \
  --target-engine-version 8.0.mysql_aurora.3.10.0
```

Wait for the Green cluster to be ready (~10-60 minutes depending on data size).

### Step 3: Perform Switchover

**⚠️ Keep the workload simulator running - do NOT stop it!**

```bash
aws rds switchover-blue-green-deployment \
  --blue-green-deployment-identifier <deployment-id>
```

### Step 4: Observe Results

Watch the workload simulator console for:
- Connection errors during switchover
- Automatic reconnection behavior
- Time to recovery

Expected downtime: **3-5 seconds** with Blue-Green plugin, **11-20 seconds** without.

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                    VPC (10.0.0.0/16)                            │
│                                                                 │
│  ┌─────────────────────────┐  ┌─────────────────────────┐    │
│  │   Availability Zone 1   │  │   Availability Zone 2   │    │
│  │                         │  │                         │    │
│  │ ┌─────────────────────┐ │  │ ┌─────────────────────┐ │    │
│  │ │ Aurora Private      │ │  │ │ Aurora Private      │ │    │
│  │ │ 10.0.1.0/24         │◄┼──┼─┤ 10.0.2.0/24         │ │    │
│  │ │ (Writer)            │ │  │ │ (Reader)            │ │    │
│  │ └─────────────────────┘ │  │ └─────────────────────┘ │    │
│  │                         │  │                         │    │
│  │ ┌─────────────────────┐ │  │                         │    │
│  │ │ EC2 Public          │ │  │                         │    │
│  │ │ 10.0.10.0/24        │ │  │                         │    │
│  │ │ (Workload Sim)      │◄┼──┼─→ Write Traffic        │    │
│  │ └─────────────────────┘ │  │                         │    │
│  └─────────────────────────┘  └─────────────────────────┘    │
└─────────────────────────────────────────────────────────────────┘
```

## Key Features

- **Production-Scale Testing**: 12,000 tables to simulate metadata overhead
- **Real-Time Monitoring**: Console output with timestamps and latency tracking
- **AWS Advanced JDBC Wrapper**: Automatic failover with Blue-Green plugin
- **Multi-AZ Deployment**: High availability configuration
- **Optional EKS Support**: Scale testing with Kubernetes for advanced scenarios

## Project Structure

```
aurora-blue-green-deployment-lab/
├── README.md                    # This file - quick start guide
├── CLAUDE.md                    # Comprehensive technical documentation
├── infrastructure/              # Pulumi infrastructure code
│   ├── vpc/                    # VPC and networking
│   ├── aurora/                 # Aurora cluster
│   ├── ec2/                    # EC2 workload simulator
│   ├── deploy.sh               # Automated deployment script
│   └── destroy.sh              # Cleanup script
├── workload-simulator/          # Java application
│   ├── src/                    # Source code
│   ├── kubernetes/             # K8s manifests (optional)
│   └── Dockerfile              # Container image
└── scripts/                     # Utility scripts
    └── init-schema.sh          # Database initialization
```

## Documentation

- **[CLAUDE.md](CLAUDE.md)** - Complete technical documentation, architecture deep-dive, and all deployment options
- **[Infrastructure README](infrastructure/README.md)** - Detailed infrastructure component documentation
- **[Workload Simulator README](workload-simulator/README.md)** - Application usage and configuration guide

## Deployment Options

### Option 1: EC2-Based (Recommended for Beginners)

- Simple command-line execution
- Real-time console output
- Easy to start, stop, and observe
- Ideal for learning Blue-Green behavior

### Option 2: EKS-Based (Advanced)

- Containerized deployment for scaled testing
- Supports multiple pod instances
- Integrated with Prometheus/Grafana monitoring
- Suitable for high-concurrency scenarios

See [CLAUDE.md](CLAUDE.md) for detailed instructions on both options.

## Technology Stack

| Component | Technology |
|-----------|-----------|
| Infrastructure as Code | Pulumi (Golang) |
| Database | Amazon Aurora MySQL 3.04 → 3.10 |
| Application | Java 17 (Amazon Corretto) |
| JDBC Driver | AWS Advanced JDBC Wrapper 2.6.6 |
| Container Orchestration | Amazon EKS (optional) |
| Monitoring | Amazon Managed Prometheus + Grafana (optional) |

## Learning Objectives

By completing this lab, you will:

- ✅ Understand Aurora Blue-Green deployment architecture
- ✅ Learn how to perform zero-downtime database upgrades
- ✅ Experience real-time workload monitoring during infrastructure changes
- ✅ Gain insights into JDBC connection failover behavior
- ✅ Practice with AWS Advanced JDBC Wrapper and Blue-Green plugin

## Cleanup

To destroy all infrastructure and avoid charges:

```bash
cd infrastructure
./destroy.sh
```

Or manually:

```bash
cd infrastructure/ec2 && pulumi destroy
cd ../aurora && pulumi destroy
cd ../vpc && pulumi destroy
```

## Cost Estimate

Running this lab for 2 hours:

- Aurora r6g.xlarge instances (2): ~$1.50
- EC2 t3.xlarge instance: ~$0.40
- Data transfer: ~$0.10
- **Total: ~$2.00 per 2-hour session**

Remember to destroy resources when not in use!

## Troubleshooting

### Connection Pool Exhausted
```
ERROR: HikariPool - Connection is not available
```
**Solution**: Increase `--connection-pool-size` to at least 10 per worker

### Aurora Cluster Not Accessible
```
ERROR: Communications link failure
```
**Solution**: Verify security group allows traffic from EC2 subnet (10.0.10.0/24)

### Schema Initialization Timeout
**Solution**: Increase `--parallel` parameter or run in smaller batches

For more troubleshooting, see [CLAUDE.md - Troubleshooting](CLAUDE.md#troubleshooting).

## Resources

- [Aurora Blue-Green Deployments Documentation](https://docs.aws.amazon.com/AmazonRDS/latest/AuroraUserGuide/blue-green-deployments.html)
- [AWS Advanced JDBC Wrapper](https://github.com/awslabs/aws-advanced-jdbc-wrapper)
- [Pulumi AWS Provider](https://www.pulumi.com/registry/packages/aws/)

## Contributing

Contributions are welcome! Please read [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Support

For issues or questions:
- Open an issue on GitHub
- Check [CLAUDE.md](CLAUDE.md) for detailed documentation
- Review [troubleshooting section](#troubleshooting)

---

**Ready to test Aurora Blue-Green deployments?** Start with the [Quick Start](#quick-start) guide above! 🚀
