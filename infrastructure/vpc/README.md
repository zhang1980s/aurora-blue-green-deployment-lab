# VPC Infrastructure

This directory contains the Pulumi code for creating the VPC and network infrastructure for the Aurora Blue-Green deployment lab.

## Architecture

The infrastructure creates:

- **VPC**: 10.0.0.0/16 CIDR block with DNS support
- **Subnets**:
  - Aurora Private Subnets: 10.0.1.0/24 (AZ1), 10.0.2.0/24 (AZ2)
  - EC2 Public Subnet: 10.0.10.0/24 (AZ1)
  - EKS Private Subnets: 10.0.20.0/24 (AZ1), 10.0.21.0/24 (AZ2)
- **Internet Gateway**: For public subnet internet access
- **Route Tables**:
  - Public route table with IGW route
  - Private route table (no internet access)
- **Security Groups**:
  - Aurora SG: MySQL port 3306 from EC2 and EKS subnets
  - EC2 SG: SSH port 22 from anywhere, all outbound
  - EKS SG: Inter-node communication, all outbound

## Prerequisites

- Pulumi CLI installed
- Go 1.21+ installed
- AWS credentials configured
- AWS CLI configured with appropriate permissions

## Deployment

1. Initialize the Pulumi stack:
   ```bash
   pulumi stack init dev
   ```

2. Configure AWS region:
   ```bash
   pulumi config set aws:region us-east-1
   ```

3. (Optional) Customize configuration:
   ```bash
   pulumi config set vpcCidr "10.0.0.0/16"
   pulumi config set projectName "aurora-bluegreen-lab"
   ```

4. Preview the infrastructure:
   ```bash
   pulumi preview
   ```

5. Deploy the infrastructure:
   ```bash
   pulumi up
   ```

## Outputs

After deployment, the following outputs are available:

- `vpcId`: VPC ID
- `vpcCidr`: VPC CIDR block
- `auroraSubnet1Id`: Aurora private subnet 1 ID
- `auroraSubnet2Id`: Aurora private subnet 2 ID
- `ec2SubnetId`: EC2 public subnet ID
- `eksSubnet1Id`: EKS private subnet 1 ID
- `eksSubnet2Id`: EKS private subnet 2 ID
- `auroraSecurityGroupId`: Aurora security group ID
- `ec2SecurityGroupId`: EC2 security group ID
- `eksSecurityGroupId`: EKS security group ID
- `availabilityZone1`: First availability zone
- `availabilityZone2`: Second availability zone

## Retrieve Outputs

```bash
pulumi stack output vpcId
pulumi stack output --json
```

## Cleanup

To destroy the infrastructure:

```bash
pulumi destroy
```
