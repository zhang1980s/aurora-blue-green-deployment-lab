# Aurora Cluster Infrastructure

This directory contains the Pulumi code for creating the Aurora MySQL cluster for the Blue-Green deployment lab.

## Architecture

The infrastructure creates:

- **Aurora MySQL Cluster**: Version 3.04 (initial) → 3.10 (target upgrade)
- **Writer Instance**: db.r6g.xlarge with Performance Insights enabled
- **Reader Instance**: db.r6g.xlarge with Performance Insights enabled
- **DB Subnet Group**: Spanning 2 private subnets in different AZs
- **Parameter Groups**: Cluster and instance-level parameter groups
- **Security**: Storage encryption enabled, CloudWatch logs enabled

## Prerequisites

- Pulumi CLI installed
- Go 1.21+ installed
- AWS credentials configured
- VPC infrastructure deployed (from `infrastructure/vpc`)

## Configuration

The Aurora cluster requires the VPC stack outputs. You must configure the VPC stack reference:

```bash
pulumi config set vpcStackName "organization/aurora-bluegreen-vpc/dev"
```

Replace with your actual VPC stack name in format: `organization/project/stack`

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

4. Set the master password (required, secret):
   ```bash
   pulumi config set --secret masterPassword "YourStrongPassword123!"
   ```

5. (Optional) Customize configuration:
   ```bash
   pulumi config set projectName "aurora-bluegreen-lab"
   pulumi config set databaseName "lab_db"
   pulumi config set masterUsername "admin"
   pulumi config set engineVersion "8.0.mysql_aurora.3.04.0"
   pulumi config set instanceClass "db.r6g.xlarge"
   ```

6. Preview the infrastructure:
   ```bash
   pulumi preview
   ```

7. Deploy the infrastructure:
   ```bash
   pulumi up
   ```

   Note: Aurora cluster creation takes approximately 10-15 minutes.

## Outputs

After deployment, the following outputs are available:

- `clusterIdentifier`: Aurora cluster identifier
- `clusterArn`: Aurora cluster ARN
- `clusterEndpoint`: Writer endpoint (use this for write operations)
- `clusterReaderEndpoint`: Reader endpoint (use this for read operations)
- `clusterPort`: Database port (default: 3306)
- `databaseName`: Name of the initial database
- `masterUsername`: Master username
- `engineVersion`: Current engine version
- `writerInstanceId`: Writer instance ID
- `readerInstanceId`: Reader instance ID
- `writerInstanceEndpoint`: Writer instance endpoint
- `readerInstanceEndpoint`: Reader instance endpoint

## Retrieve Outputs

```bash
# Get cluster endpoint
pulumi stack output clusterEndpoint

# Get all outputs as JSON
pulumi stack output --json

# Example: Connect to the database
mysql -h $(pulumi stack output clusterEndpoint) -u admin -p lab_db
```

## Post-Deployment: Initialize Schema

After deploying the Aurora cluster, run the schema initialization script to create 12,000 tables:

```bash
cd ../../scripts
./init-schema.sh
```

This process may take 30-60 minutes to complete.

## Blue-Green Deployment Process

Once the cluster is deployed and schema initialized:

1. Start the workload simulator (see `infrastructure/ec2` or `workload-simulator`)
2. Create a Blue-Green deployment via AWS Console or CLI:
   ```bash
   aws rds create-blue-green-deployment \
     --blue-green-deployment-name aurora-upgrade-test \
     --source-arn $(pulumi stack output clusterArn) \
     --target-engine-version 8.0.mysql_aurora.3.10.0
   ```
3. Wait for the Green environment to be ready
4. Perform the switchover when ready:
   ```bash
   aws rds switchover-blue-green-deployment \
     --blue-green-deployment-identifier <deployment-id>
   ```

## Cleanup

To destroy the infrastructure:

```bash
pulumi destroy
```

Note: Ensure you have backups if needed before destroying the cluster.
