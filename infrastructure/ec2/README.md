# EC2 Workload Simulator Infrastructure

This directory contains the Pulumi code for creating the EC2 instance that hosts the workload simulator for the Aurora Blue-Green deployment lab.

## Architecture

The infrastructure creates:

- **EC2 Instance**: t3.xlarge with Amazon Linux 2023
- **Pre-installed Software**:
  - Amazon Corretto 17 (OpenJDK)
  - MySQL client for database testing
  - Git for repository operations
- **Workload Simulator Directory**: `/opt/workload-simulator`
  - Helper scripts for easy execution
  - README with usage instructions
- **Security**:
  - Deployed in public subnet with public IP
  - SSH access via key pair
  - Outbound access to Aurora private subnets
  - Root volume encrypted with GP3 storage

## Prerequisites

- Pulumi CLI installed
- Go 1.21+ installed
- AWS credentials configured
- VPC infrastructure deployed (from `infrastructure/vpc`)
- Aurora cluster deployed (from `infrastructure/aurora`)
- **AWS EC2 Key Pair created** in the target region

## Create EC2 Key Pair

If you don't have an EC2 key pair, create one first:

```bash
# Create a new key pair
aws ec2 create-key-pair \
  --key-name aurora-lab-key \
  --query 'KeyMaterial' \
  --output text > aurora-lab-key.pem

# Set proper permissions
chmod 400 aurora-lab-key.pem
```

## Deployment

1. Initialize the Pulumi stack:
   ```bash
   pulumi stack init dev
   ```

2. Configure AWS region (must match VPC region):
   ```bash
   pulumi config set aws:region us-east-1
   ```

3. Configure the VPC stack reference:
   ```bash
   pulumi config set vpcStackName "organization/aurora-bluegreen-vpc/dev"
   ```

4. Configure your EC2 key pair name (required):
   ```bash
   pulumi config set keyName "aurora-lab-key"
   ```

5. (Optional) Configure Aurora stack reference for convenience:
   ```bash
   pulumi config set auroraStackName "organization/aurora-bluegreen-aurora/dev"
   ```

6. (Optional) Customize configuration:
   ```bash
   pulumi config set projectName "aurora-bluegreen-lab"
   pulumi config set instanceType "t3.xlarge"
   ```

7. Preview the infrastructure:
   ```bash
   pulumi preview
   ```

8. Deploy the infrastructure:
   ```bash
   pulumi up
   ```

   Note: EC2 instance creation takes approximately 2-3 minutes.

## Outputs

After deployment, the following outputs are available:

- `instanceId`: EC2 instance ID
- `publicIp`: Public IP address
- `publicDns`: Public DNS name
- `privateIp`: Private IP address
- `instanceType`: Instance type
- `availabilityZone`: Availability zone
- `sshCommand`: Ready-to-use SSH command
- `workloadSimulatorPath`: Path to workload simulator directory
- `auroraClusterEndpoint`: (If configured) Aurora cluster endpoint
- `runSimulatorCommand`: (If configured) Ready-to-use command to run the simulator

## Retrieve Outputs

```bash
# Get SSH command
pulumi stack output sshCommand

# Get public IP
pulumi stack output publicIp

# Get all outputs as JSON
pulumi stack output --json
```

## Post-Deployment: Upload Workload Simulator

After building the workload simulator JAR, upload it to the EC2 instance:

```bash
# Build the workload simulator (from project root)
cd workload-simulator
mvn clean package

# Upload to EC2 instance
scp -i aurora-lab-key.pem \
  target/workload-simulator.jar \
  ec2-user@$(pulumi stack output publicIp):/opt/workload-simulator/
```

## Connect to EC2 Instance

```bash
# Use the SSH command from outputs
$(pulumi stack output sshCommand)

# Or manually
ssh -i aurora-lab-key.pem ec2-user@$(pulumi stack output publicIp)
```

## Run Workload Simulator

Once connected to the EC2 instance:

### Method 1: Using Helper Script (Recommended)

```bash
cd /opt/workload-simulator

# Run with default settings
./run-simulator.sh <your-aurora-endpoint>

# Run with custom settings
./run-simulator.sh <your-aurora-endpoint> \
  --write-workers 20 \
  --write-rate 200 \
  --connection-pool-size 200
```

### Method 2: Direct Java Execution

```bash
cd /opt/workload-simulator

java -jar workload-simulator.jar \
  --aurora-endpoint <your-aurora-endpoint> \
  --database-name lab_db \
  --write-workers 10 \
  --write-rate 100 \
  --connection-pool-size 100
```

### Method 3: Using Pulumi Output (If Aurora Stack Configured)

```bash
# From your local machine, get the command
pulumi stack output runSimulatorCommand

# SSH into the instance and run that command
ssh -i aurora-lab-key.pem ec2-user@$(pulumi stack output publicIp)
cd /opt/workload-simulator
$(pulumi stack output runSimulatorCommand -s ../aurora/dev)
```

## Testing Aurora Connection

Before running the workload simulator, test the Aurora connection:

```bash
# Test MySQL connectivity
mysql -h <aurora-endpoint> -u admin -p lab_db

# If connection succeeds, you're ready to run the workload simulator
```

## Monitoring During Blue-Green Deployment

1. SSH into the EC2 instance
2. Start the workload simulator
3. Observe the console output (success/failure logs)
4. In another terminal, monitor CloudWatch or AWS Console
5. Initiate the Blue-Green deployment
6. Watch the workload simulator logs during switchover
7. Verify recovery after switchover completes

## Troubleshooting

### Cannot connect via SSH

- Check that your security group allows inbound SSH from your IP
- Verify the key pair permissions: `chmod 400 your-key.pem`
- Ensure the instance is in a public subnet with IGW route

### Cannot connect to Aurora

- Verify Aurora security group allows connections from EC2 subnet (10.0.10.0/24)
- Check that Aurora endpoint is correct
- Ensure Aurora cluster is available and running

### Workload simulator JAR not found

- Upload the JAR using the SCP command shown above
- Verify the file exists: `ls -la /opt/workload-simulator/`

## Cleanup

To destroy the infrastructure:

```bash
pulumi destroy
```

Note: This will terminate the EC2 instance and delete associated resources.
