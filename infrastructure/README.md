# Infrastructure

This directory contains the Infrastructure as Code (IaC) for the Aurora Blue-Green Deployment Lab using Pulumi and Go.

## Architecture Overview

The infrastructure is organized into three separate Pulumi stacks that must be deployed in order:

1. **VPC** (`vpc/`): Network infrastructure including VPC, subnets, security groups, and routing
2. **Aurora** (`aurora/`): Aurora MySQL cluster with writer and reader instances
3. **EC2** (`ec2/`): EC2 instance for hosting the workload simulator

```
┌─────────────────────────────────────────────────────────────────┐
│                    VPC (10.0.0.0/16)                            │
│                                                                 │
│  ┌─────────────────────────┐  ┌─────────────────────────┐    │
│  │   Availability Zone 1   │  │   Availability Zone 2   │    │
│  │                         │  │                         │    │
│  │ ┌─────────────────────┐ │  │ ┌─────────────────────┐ │    │
│  │ │ Aurora Private      │ │  │ │ Aurora Private      │ │    │
│  │ │ 10.0.1.0/24         │ │  │ │ 10.0.2.0/24         │ │    │
│  │ │ ┌─────────────────┐ │ │  │ │ ┌─────────────────┐ │ │    │
│  │ │ │ Writer Instance │ │ │  │ │ │ Reader Instance │ │ │    │
│  │ │ │ r6g.xlarge      │ │ │  │ │ │ r6g.xlarge      │ │ │    │
│  │ │ └─────────────────┘ │ │  │ │ └─────────────────┘ │ │    │
│  │ └─────────────────────┘ │  │ └─────────────────────┘ │    │
│  │                         │  │                         │    │
│  │ ┌─────────────────────┐ │  │                         │    │
│  │ │ EC2 Public          │ │  │                         │    │
│  │ │ 10.0.10.0/24        │←┼──┼─→ Internet Gateway     │    │
│  │ │ ┌─────────────────┐ │ │  │                         │    │
│  │ │ │ EC2 t3.xlarge   │ │ │  │                         │    │
│  │ │ │ Workload Sim    │ │ │  │                         │    │
│  │ │ └─────────────────┘ │ │  │                         │    │
│  │ └─────────────────────┘ │  │                         │    │
│  │                         │  │                         │    │
│  │ ┌─────────────────────┐ │  │ ┌─────────────────────┐ │    │
│  │ │ EKS Private         │ │  │ │ EKS Private         │ │    │
│  │ │ 10.0.20.0/24        │ │  │ │ 10.0.21.0/24        │ │    │
│  │ │ (Optional)          │ │  │ │ (Optional)          │ │    │
│  │ └─────────────────────┘ │  │ └─────────────────────┘ │    │
│  └─────────────────────────┘  └─────────────────────────┘    │
└─────────────────────────────────────────────────────────────────┘
```

## Prerequisites

### Required Tools

- **Pulumi CLI** (v3.x or later): [Installation Guide](https://www.pulumi.com/docs/get-started/install/)
- **Go** (1.21 or later): [Installation Guide](https://go.dev/doc/install)
- **AWS CLI** (configured): [Installation Guide](https://aws.amazon.com/cli/)
- **AWS Credentials**: Configured via `aws configure` or environment variables

### AWS Permissions Required

Your AWS credentials need the following permissions:
- EC2: VPC, Subnet, Security Group, Internet Gateway, Route Table management
- RDS: Aurora cluster and instance creation, parameter groups, subnet groups
- EC2: Instance creation, AMI lookup, key pair usage
- IAM: Role creation (if using EKS - optional)
- CloudWatch: Log group creation

### AWS Resources to Create

Before deploying, create an EC2 key pair in your target region:

```bash
aws ec2 create-key-pair \
  --key-name aurora-lab-key \
  --query 'KeyMaterial' \
  --output text > aurora-lab-key.pem

chmod 400 aurora-lab-key.pem
```

## Quick Start

### Step 1: Deploy VPC Infrastructure

```bash
cd vpc

# Initialize stack
pulumi stack init dev

# Configure region
pulumi config set aws:region us-east-1

# (Optional) Customize configuration
pulumi config set vpcCidr "10.0.0.0/16"
pulumi config set projectName "aurora-bluegreen-lab"

# Deploy
pulumi up

# Save the stack name for later reference
export VPC_STACK="$(pulumi whoami)/aurora-bluegreen-vpc/dev"
```

### Step 2: Deploy Aurora Cluster

```bash
cd ../aurora

# Initialize stack
pulumi stack init dev

# Configure region (must match VPC)
pulumi config set aws:region us-east-1

# Reference VPC stack
pulumi config set vpcStackName "$VPC_STACK"

# Set master password (required, secret)
pulumi config set --secret masterPassword "YourStrongPassword123!"

# (Optional) Customize configuration
pulumi config set engineVersion "8.0.mysql_aurora.3.04.0"
pulumi config set instanceClass "db.r6g.xlarge"

# Deploy
pulumi up

# Save Aurora endpoint for later use
export AURORA_ENDPOINT="$(pulumi stack output clusterEndpoint)"

# Save the stack name for later reference
export AURORA_STACK="$(pulumi whoami)/aurora-bluegreen-aurora/dev"
```

**Important**: Aurora cluster creation takes approximately 10-15 minutes.

### Step 3: Initialize Database Schema

```bash
# Navigate to scripts directory
cd ../../scripts

# Run schema initialization (creates 12,000 tables)
./init-schema.sh

# This process takes 30-60 minutes
# Progress is logged to schema-init.log
```

### Step 4: Deploy EC2 Workload Simulator

```bash
cd ../infrastructure/ec2

# Initialize stack
pulumi stack init dev

# Configure region (must match VPC)
pulumi config set aws:region us-east-1

# Reference VPC stack
pulumi config set vpcStackName "$VPC_STACK"

# (Optional) Reference Aurora stack for convenience outputs
pulumi config set auroraStackName "$AURORA_STACK"

# Set EC2 key pair name
pulumi config set keyName "aurora-lab-key"

# Deploy
pulumi up

# Get SSH command
export SSH_CMD="$(pulumi stack output sshCommand)"
```

### Step 5: Upload and Run Workload Simulator

```bash
# Build the workload simulator
cd ../../workload-simulator
mvn clean package

# Upload to EC2
scp -i ~/aurora-lab-key.pem \
  target/workload-simulator.jar \
  ec2-user@$(cd ../infrastructure/ec2 && pulumi stack output publicIp):/opt/workload-simulator/

# SSH into EC2 instance
eval $(cd ../infrastructure/ec2 && pulumi stack output sshCommand)

# On the EC2 instance, run the simulator
cd /opt/workload-simulator
./run-simulator.sh $AURORA_ENDPOINT
```

## Component Details

### 1. VPC Infrastructure

**Location**: `vpc/`

**Creates**:
- VPC with configurable CIDR (default: 10.0.0.0/16)
- 2 private subnets for Aurora (10.0.1.0/24, 10.0.2.0/24)
- 1 public subnet for EC2 (10.0.10.0/24)
- 2 private subnets for EKS (10.0.20.0/24, 10.0.21.0/24) - optional
- Internet Gateway for public subnet
- Route tables and associations
- Security groups for Aurora, EC2, and EKS

**Key Outputs**:
- `vpcId`, `auroraSubnet1Id`, `auroraSubnet2Id`, `ec2SubnetId`
- `auroraSecurityGroupId`, `ec2SecurityGroupId`, `eksSecurityGroupId`

[Full VPC Documentation](vpc/README.md)

### 2. Aurora Cluster

**Location**: `aurora/`

**Creates**:
- Aurora MySQL cluster (version 3.04 for upgrade testing)
- 1 writer instance (db.r6g.xlarge)
- 1 reader instance (db.r6g.xlarge)
- DB subnet group spanning 2 AZs
- Cluster and instance parameter groups
- CloudWatch log exports (error, general, slowquery)
- Performance Insights enabled

**Key Outputs**:
- `clusterEndpoint` (writer endpoint)
- `clusterReaderEndpoint` (reader endpoint)
- `clusterArn`, `databaseName`, `engineVersion`

**Important**: Requires VPC stack outputs for subnet and security group references.

[Full Aurora Documentation](aurora/README.md)

### 3. EC2 Workload Simulator

**Location**: `ec2/`

**Creates**:
- EC2 instance (t3.xlarge) with Amazon Linux 2023
- Pre-installed Amazon Corretto 17 (OpenJDK)
- MySQL client and git
- Workload simulator directory at `/opt/workload-simulator`
- Helper scripts for easy execution
- 30GB encrypted GP3 root volume

**Key Outputs**:
- `publicIp`, `publicDns`, `privateIp`
- `sshCommand` (ready-to-use SSH command)
- `auroraClusterEndpoint` (if configured)

**Important**: Requires VPC stack outputs and an existing EC2 key pair.

[Full EC2 Documentation](ec2/README.md)

## Configuration Reference

### VPC Configuration

```bash
pulumi config set vpcCidr "10.0.0.0/16"          # VPC CIDR block
pulumi config set projectName "my-project"        # Project name for tagging
```

### Aurora Configuration

```bash
pulumi config set vpcStackName "org/vpc/dev"               # VPC stack reference (required)
pulumi config set --secret masterPassword "Pass123!"        # Master password (required)
pulumi config set databaseName "lab_db"                     # Database name
pulumi config set masterUsername "admin"                    # Master username
pulumi config set engineVersion "8.0.mysql_aurora.3.04.0"   # Engine version
pulumi config set instanceClass "db.r6g.xlarge"             # Instance class
```

### EC2 Configuration

```bash
pulumi config set vpcStackName "org/vpc/dev"          # VPC stack reference (required)
pulumi config set keyName "my-key"                     # EC2 key pair (required)
pulumi config set auroraStackName "org/aurora/dev"    # Aurora stack reference (optional)
pulumi config set instanceType "t3.xlarge"             # Instance type
```

## Stack References

Pulumi uses stack references to share outputs between stacks. The format is:

```
<organization>/<project>/<stack>
```

To get your stack name:

```bash
cd vpc
pulumi whoami              # Get your organization
pulumi stack ls            # List stacks
echo "$(pulumi whoami)/aurora-bluegreen-vpc/dev"
```

## Managing Pulumi Stacks

### View Stack Outputs

```bash
cd vpc
pulumi stack output                    # Show all outputs
pulumi stack output vpcId              # Show specific output
pulumi stack output --json             # JSON format
```

### Update Configuration

```bash
pulumi config set keyName "new-key"    # Update configuration
pulumi config                          # View all configuration
```

### Update Infrastructure

```bash
pulumi preview                         # Preview changes
pulumi up                              # Apply changes
pulumi refresh                         # Sync state with AWS
```

### Destroy Infrastructure

**Important**: Destroy in reverse order:

```bash
# 1. Destroy EC2 first
cd ec2
pulumi destroy

# 2. Destroy Aurora second
cd ../aurora
pulumi destroy

# 3. Destroy VPC last
cd ../vpc
pulumi destroy
```

## Cost Estimation

Approximate monthly costs (us-east-1 region):

| Resource | Configuration | Monthly Cost |
|----------|--------------|--------------|
| Aurora Writer | db.r6g.xlarge | ~$365 |
| Aurora Reader | db.r6g.xlarge | ~$365 |
| EC2 Instance | t3.xlarge (24/7) | ~$150 |
| Storage | 100GB Aurora + 30GB EBS | ~$15 |
| Data Transfer | Minimal | ~$5 |
| **Total** | | **~$900/month** |

**Cost Optimization Tips**:
- Stop EC2 instance when not testing (saves ~$150/month)
- Use Aurora Serverless v2 for non-production testing
- Delete the infrastructure after completing the lab
- Use smaller instance types for initial testing (e.g., db.t3.medium, t3.medium)

## Troubleshooting

### Pulumi Login Issues

```bash
# Login to Pulumi Cloud
pulumi login

# Or use local backend
pulumi login file://~/.pulumi
```

### Stack Reference Errors

```bash
# Verify stack exists
pulumi stack ls

# Check stack outputs
cd vpc
pulumi stack output --json

# Verify stack name format
echo "$(pulumi whoami)/aurora-bluegreen-vpc/dev"
```

### AWS Credentials

```bash
# Verify AWS credentials
aws sts get-caller-identity

# Configure AWS profile
export AWS_PROFILE=your-profile
pulumi config set aws:profile your-profile
```

### Dependency Errors

Ensure you deploy in order:
1. VPC first (no dependencies)
2. Aurora second (depends on VPC)
3. EC2 last (depends on VPC, optionally references Aurora)

### Go Module Issues

```bash
cd vpc  # or aurora, ec2
go mod tidy
go mod download
```

## Testing the Blue-Green Deployment

Once all infrastructure is deployed:

1. **Start Workload Simulator**:
   ```bash
   ssh -i aurora-lab-key.pem ec2-user@<ec2-public-ip>
   cd /opt/workload-simulator
   ./run-simulator.sh <aurora-endpoint>
   ```

2. **Create Blue-Green Deployment**:
   ```bash
   aws rds create-blue-green-deployment \
     --blue-green-deployment-name aurora-upgrade-test \
     --source-arn <aurora-cluster-arn> \
     --target-engine-version 8.0.mysql_aurora.3.10.0
   ```

3. **Monitor Workload**:
   - Watch console output for success/failure logs
   - Observe connection behavior during switchover

4. **Perform Switchover**:
   ```bash
   aws rds switchover-blue-green-deployment \
     --blue-green-deployment-identifier <deployment-id>
   ```

5. **Validate**:
   - Verify workload resumes successfully
   - Check Aurora version: `SELECT @@aurora_version;`

## Additional Resources

- [Pulumi AWS Provider Documentation](https://www.pulumi.com/registry/packages/aws/)
- [Aurora Blue-Green Deployments](https://docs.aws.amazon.com/AmazonRDS/latest/AuroraUserGuide/blue-green-deployments.html)
- [Amazon Aurora MySQL Version 3.10 Release Notes](https://docs.aws.amazon.com/AmazonRDS/latest/AuroraMySQLReleaseNotes/AuroraMySQL.Updates.3010.html)

## Support

For issues or questions:
- Check the README files in each component directory
- Review Pulumi logs: `pulumi logs`
- Check AWS CloudWatch logs for Aurora and EC2
- Review the main project [CLAUDE.md](../CLAUDE.md) documentation
