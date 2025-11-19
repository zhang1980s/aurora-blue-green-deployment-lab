# Quick Start Guide

This guide will help you deploy the Aurora Blue-Green deployment lab infrastructure in minutes.

## Prerequisites Checklist

- [ ] Pulumi CLI installed ([Install Guide](https://www.pulumi.com/docs/get-started/install/))
- [ ] Go 1.21+ installed ([Install Guide](https://go.dev/doc/install))
- [ ] AWS CLI installed and configured ([Install Guide](https://aws.amazon.com/cli/))
- [ ] AWS credentials configured (`aws configure`)
- [ ] EC2 key pair created in target region

## Create EC2 Key Pair

```bash
# Create key pair
aws ec2 create-key-pair \
  --key-name aurora-lab-key \
  --query 'KeyMaterial' \
  --output text > aurora-lab-key.pem

# Set permissions
chmod 400 aurora-lab-key.pem
```

## Option 1: Automated Deployment (Recommended)

Use the automated deployment script:

```bash
cd infrastructure
./deploy.sh
```

The script will:
1. Check prerequisites
2. Prompt for configuration (region, project name, passwords, etc.)
3. Deploy VPC infrastructure
4. Deploy Aurora cluster (10-15 minutes)
5. Optionally initialize database schema (30-60 minutes)
6. Deploy EC2 workload simulator

## Option 2: Manual Deployment

### Step 1: Deploy VPC (5 minutes)

```bash
cd infrastructure/vpc

pulumi stack init dev
pulumi config set aws:region us-east-1
pulumi config set projectName "aurora-bluegreen-lab"

pulumi up --yes

# Save stack name for later
export VPC_STACK="$(pulumi whoami)/aurora-bluegreen-vpc/dev"
cd ..
```

### Step 2: Deploy Aurora (10-15 minutes)

```bash
cd aurora

pulumi stack init dev
pulumi config set aws:region us-east-1
pulumi config set vpcStackName "$VPC_STACK"
pulumi config set --secret masterPassword "YourStrongPassword123!"

pulumi up --yes

# Save outputs
export AURORA_ENDPOINT="$(pulumi stack output clusterEndpoint)"
export AURORA_STACK="$(pulumi whoami)/aurora-bluegreen-aurora/dev"
cd ..
```

### Step 3: Initialize Schema (30-60 minutes) - Optional but Recommended

```bash
cd ../scripts
./init-schema.sh
cd ../infrastructure
```

### Step 4: Deploy EC2 (5 minutes)

```bash
cd ec2

pulumi stack init dev
pulumi config set aws:region us-east-1
pulumi config set vpcStackName "$VPC_STACK"
pulumi config set auroraStackName "$AURORA_STACK"
pulumi config set keyName "aurora-lab-key"

pulumi up --yes

# Get connection info
export EC2_IP="$(pulumi stack output publicIp)"
export SSH_CMD="$(pulumi stack output sshCommand)"
cd ..
```

## Upload and Run Workload Simulator

### Build the Simulator

```bash
cd ../workload-simulator
mvn clean package
```

### Upload to EC2

```bash
scp -i aurora-lab-key.pem \
  target/workload-simulator.jar \
  ec2-user@$EC2_IP:/opt/workload-simulator/
```

### Connect and Run

```bash
# SSH into EC2
ssh -i aurora-lab-key.pem ec2-user@$EC2_IP

# Run simulator
cd /opt/workload-simulator
./run-simulator.sh <your-aurora-endpoint>
```

## Test Blue-Green Deployment

### 1. Start Workload

In the EC2 instance:
```bash
cd /opt/workload-simulator
./run-simulator.sh <aurora-endpoint>
```

### 2. Create Blue-Green Deployment

In another terminal on your local machine:
```bash
aws rds create-blue-green-deployment \
  --blue-green-deployment-name aurora-upgrade-test \
  --source-arn <aurora-cluster-arn> \
  --target-engine-version 8.0.mysql_aurora.3.10.0 \
  --region us-east-1
```

### 3. Monitor Deployment

```bash
# Check deployment status
aws rds describe-blue-green-deployments \
  --blue-green-deployment-identifier <deployment-id> \
  --region us-east-1
```

### 4. Perform Switchover

When the green environment is ready:
```bash
aws rds switchover-blue-green-deployment \
  --blue-green-deployment-identifier <deployment-id> \
  --region us-east-1
```

### 5. Observe Results

- Watch the workload simulator console for connection errors
- Note the timing of any failures
- Verify automatic reconnection
- Confirm workload resumes successfully

## Get Stack Information

```bash
# VPC outputs
cd infrastructure/vpc
pulumi stack output --json

# Aurora outputs
cd ../aurora
pulumi stack output clusterEndpoint
pulumi stack output clusterArn

# EC2 outputs
cd ../ec2
pulumi stack output publicIp
pulumi stack output sshCommand
```

## Clean Up

When you're done testing:

```bash
cd infrastructure
./destroy.sh
```

Or manually:

```bash
# Destroy in reverse order
cd infrastructure/ec2
pulumi destroy

cd ../aurora
pulumi destroy

cd ../vpc
pulumi destroy
```

## Troubleshooting

### "Stack not found" error

```bash
pulumi stack ls
# Verify stack exists

pulumi whoami
# Get your organization name

# Format: <org>/aurora-bluegreen-vpc/dev
export VPC_STACK="$(pulumi whoami)/aurora-bluegreen-vpc/dev"
```

### AWS credentials error

```bash
aws sts get-caller-identity
# Should show your AWS account

aws configure
# Reconfigure if needed
```

### Cannot connect to Aurora

```bash
# Test from EC2 instance
mysql -h <aurora-endpoint> -u admin -p lab_db

# Check security groups allow 3306 from EC2 subnet
```

### Key pair error

```bash
# List key pairs in region
aws ec2 describe-key-pairs --region us-east-1

# Create new key pair if needed
aws ec2 create-key-pair \
  --key-name aurora-lab-key \
  --query 'KeyMaterial' \
  --output text > aurora-lab-key.pem
chmod 400 aurora-lab-key.pem
```

## Cost Estimation

| Component | Cost/Hour | Cost/Day | Cost/Month |
|-----------|-----------|----------|------------|
| Aurora (2x r6g.xlarge) | ~$1.00 | ~$24 | ~$730 |
| EC2 (t3.xlarge) | ~$0.20 | ~$4.80 | ~$150 |
| Storage & Transfer | ~$0.01 | ~$0.20 | ~$20 |
| **Total** | **~$1.21** | **~$29** | **~$900** |

**💡 Cost Saving Tips:**
- Stop EC2 when not testing
- Delete infrastructure after completing the lab
- Use smaller instances for initial testing

## Next Steps

1. ✅ Complete the quick start deployment
2. 📚 Read the detailed [README.md](README.md) for advanced configurations
3. 🧪 Test the Blue-Green deployment process
4. 📊 Optional: Set up monitoring with EKS + Prometheus/Grafana
5. 🧹 Clean up resources when done

## Support

- Check individual component READMEs in `vpc/`, `aurora/`, `ec2/` directories
- Review [main project documentation](../CLAUDE.md)
- Check AWS CloudWatch logs for troubleshooting
