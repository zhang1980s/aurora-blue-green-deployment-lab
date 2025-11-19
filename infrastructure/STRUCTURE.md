# Infrastructure Directory Structure

This document provides an overview of the infrastructure code organization.

```
infrastructure/
├── README.md                           # Main infrastructure documentation
├── QUICKSTART.md                       # Quick start guide for deployment
├── STRUCTURE.md                        # This file - directory structure overview
├── Makefile                            # Convenient make targets for infrastructure management
├── deploy.sh                           # Automated deployment script
├── destroy.sh                          # Automated destruction script
├── .gitignore                          # Git ignore patterns for Pulumi and Go
│
├── vpc/                                # VPC and network infrastructure
│   ├── main.go                         # Pulumi Go code for VPC, subnets, security groups
│   ├── go.mod                          # Go module definition
│   ├── Pulumi.yaml                     # Pulumi project definition
│   ├── Pulumi.dev.example.yaml        # Example stack configuration
│   └── README.md                       # VPC deployment documentation
│
├── aurora/                             # Aurora MySQL cluster
│   ├── main.go                         # Pulumi Go code for Aurora cluster and instances
│   ├── go.mod                          # Go module definition
│   ├── Pulumi.yaml                     # Pulumi project definition
│   ├── Pulumi.dev.example.yaml        # Example stack configuration
│   └── README.md                       # Aurora deployment documentation
│
└── ec2/                                # EC2 workload simulator
    ├── main.go                         # Pulumi Go code for EC2 instance
    ├── go.mod                          # Go module definition
    ├── Pulumi.yaml                     # Pulumi project definition
    ├── Pulumi.dev.example.yaml        # Example stack configuration
    └── README.md                       # EC2 deployment documentation
```

## File Descriptions

### Root Level Files

| File | Purpose |
|------|---------|
| **README.md** | Comprehensive documentation covering architecture, deployment steps, configuration, and troubleshooting |
| **QUICKSTART.md** | Fast-track deployment guide for users who want to get started quickly |
| **STRUCTURE.md** | This file - explains the directory organization |
| **Makefile** | Provides convenient `make` commands for common operations (deploy, destroy, outputs, etc.) |
| **deploy.sh** | Interactive script that automates the entire deployment process |
| **destroy.sh** | Interactive script that safely destroys infrastructure in the correct order |
| **.gitignore** | Prevents committing Pulumi state, Go build artifacts, and IDE files |

### Component Directories

Each component (vpc, aurora, ec2) follows the same structure:

| File | Purpose |
|------|---------|
| **main.go** | Pulumi IaC code in Go defining the infrastructure resources |
| **go.mod** | Go module file declaring dependencies |
| **Pulumi.yaml** | Pulumi project definition with configurable parameters |
| **Pulumi.dev.example.yaml** | Example configuration showing all available settings |
| **README.md** | Component-specific documentation with deployment instructions |

## Infrastructure Dependencies

```
┌─────────────────────────────────────────────────┐
│                                                 │
│  VPC Infrastructure (vpc/)                      │
│  - VPC, Subnets, Security Groups                │
│  - Internet Gateway, Route Tables               │
│  - Foundation for all other components          │
│                                                 │
└─────────────┬───────────────────────────────────┘
              │
              ├─────────────────────────────────────┐
              │                                     │
              ▼                                     ▼
┌─────────────────────────────┐   ┌─────────────────────────────┐
│                             │   │                             │
│  Aurora Cluster (aurora/)   │   │  EC2 Instance (ec2/)        │
│  - References VPC outputs   │   │  - References VPC outputs   │
│  - Uses private subnets     │◄──┤  - References Aurora outputs│
│  - Database infrastructure  │   │  - Workload simulator host  │
│                             │   │                             │
└─────────────────────────────┘   └─────────────────────────────┘
```

## Deployment Order

Infrastructure must be deployed in the following order due to dependencies:

1. **VPC** (no dependencies)
   - Creates network foundation
   - Exports subnet IDs and security group IDs

2. **Aurora** (depends on VPC)
   - References VPC stack outputs via Pulumi stack references
   - Deployed into VPC's private subnets
   - Exports cluster endpoint and credentials

3. **EC2** (depends on VPC, optionally references Aurora)
   - References VPC stack outputs for subnet and security group
   - Optionally references Aurora stack for convenience outputs
   - Deployed into VPC's public subnet

## Stack References

Components communicate via Pulumi stack references. The format is:

```
<organization>/<project-name>/<stack-name>
```

Example:
```
myorg/aurora-bluegreen-vpc/dev
myorg/aurora-bluegreen-aurora/dev
myorg/aurora-bluegreen-ec2/dev
```

### How Stack References Work

When Aurora stack needs VPC subnet IDs:

```go
// In aurora/main.go
vpcStackRef, _ := pulumi.NewStackReference(ctx, vpcStackName, nil)
auroraSubnet1Id := vpcStackRef.GetStringOutput(pulumi.String("auroraSubnet1Id"))
```

This reads the `auroraSubnet1Id` output from the VPC stack without hardcoding values.

## Generated Files (Not in Git)

The following files are generated during deployment and excluded from Git:

```
infrastructure/
├── vpc/
│   ├── .pulumi/              # Pulumi state (managed by Pulumi Cloud or local backend)
│   ├── Pulumi.dev.yaml       # Actual stack configuration (may contain secrets)
│   └── go.sum                # Go module checksums
├── aurora/
│   ├── .pulumi/
│   ├── Pulumi.dev.yaml
│   └── go.sum
└── ec2/
    ├── .pulumi/
    ├── Pulumi.dev.yaml
    └── go.sum
```

## Outputs Flow

Each component exports outputs that can be consumed by other components or users:

### VPC Outputs
- `vpcId`, `vpcCidr`
- `auroraSubnet1Id`, `auroraSubnet2Id`
- `ec2SubnetId`, `eksSubnet1Id`, `eksSubnet2Id`
- `auroraSecurityGroupId`, `ec2SecurityGroupId`, `eksSecurityGroupId`
- `availabilityZone1`, `availabilityZone2`

### Aurora Outputs
- `clusterEndpoint`, `clusterReaderEndpoint`
- `clusterArn`, `clusterIdentifier`
- `databaseName`, `masterUsername`
- `engineVersion`
- Instance IDs and endpoints

### EC2 Outputs
- `instanceId`, `publicIp`, `publicDns`, `privateIp`
- `sshCommand` (ready-to-use SSH command)
- `auroraClusterEndpoint` (if Aurora stack referenced)
- `runSimulatorCommand` (if Aurora stack referenced)

## Resource Naming Convention

All resources follow a consistent naming pattern:

```
{projectName}-{resourceType}-{suffix}
```

Examples:
- `aurora-bluegreen-lab-vpc`
- `aurora-bluegreen-lab-aurora-cluster`
- `aurora-bluegreen-lab-writer-instance`
- `aurora-bluegreen-lab-workload-simulator`

This makes resources easy to identify in the AWS Console and CloudFormation.

## Tagging Strategy

All resources are tagged with:
- `Name`: Human-readable resource name
- `Project`: Project name (e.g., "aurora-bluegreen-lab")
- `Role`: Resource role (e.g., "writer", "reader", "workload-simulator")

Additional component-specific tags:
- VPC subnets: `Type` (e.g., "private-aurora", "public-ec2", "private-eks")

## Configuration Management

Configuration is managed at three levels:

1. **Project Level** (Pulumi.yaml)
   - Default values for all stacks
   - Parameter definitions and types
   - Project metadata

2. **Stack Level** (Pulumi.dev.yaml)
   - Environment-specific values
   - Secrets (encrypted)
   - Stack-specific overrides

3. **Environment Variables**
   - AWS credentials
   - AWS region
   - Pulumi backend configuration

## Security Considerations

### Secrets Management
- Database passwords stored as Pulumi secrets (encrypted)
- Never committed to Git
- Set via `pulumi config set --secret`

### Network Isolation
- Aurora in private subnets (no internet access)
- EC2 in public subnet (SSH access only)
- Security groups restrict traffic flow
- No default 0.0.0.0/0 ingress rules (except SSH to EC2)

### Encryption
- Aurora storage encryption enabled
- EC2 root volume encrypted
- Secrets encrypted in Pulumi state

## Usage Examples

### View outputs
```bash
cd infrastructure/vpc
pulumi stack output --json
```

### Update configuration
```bash
cd infrastructure/aurora
pulumi config set instanceClass "db.r6g.2xlarge"
pulumi up
```

### Reference outputs in scripts
```bash
export AURORA_ENDPOINT=$(cd infrastructure/aurora && pulumi stack output clusterEndpoint)
export EC2_IP=$(cd infrastructure/ec2 && pulumi stack output publicIp)
```

### Use Makefile
```bash
cd infrastructure
make outputs STACK_NAME=dev
make deploy
make destroy
```

## Extending the Infrastructure

To add new components:

1. Create a new directory (e.g., `eks/`, `monitoring/`)
2. Follow the same structure (main.go, go.mod, Pulumi.yaml, README.md)
3. Reference existing stacks via `pulumi.NewStackReference()`
4. Export outputs for other components to consume
5. Update deploy.sh and destroy.sh to include the new component
6. Document the component in this file

## Maintenance

### Update Dependencies
```bash
cd vpc
go get -u github.com/pulumi/pulumi-aws/sdk/v6
go mod tidy

cd ../aurora
go get -u github.com/pulumi/pulumi-aws/sdk/v6
go mod tidy

cd ../ec2
go get -u github.com/pulumi/pulumi-aws/sdk/v6
go mod tidy
```

### Refresh State
```bash
cd infrastructure
make refresh STACK_NAME=dev
```

## Documentation Hierarchy

1. **QUICKSTART.md** - Start here for rapid deployment
2. **README.md** - Comprehensive guide with all details
3. **Component READMEs** - Component-specific documentation
4. **STRUCTURE.md** - This file - understand the organization
5. **Example configs** - Reference for configuration format

## Related Documentation

- Main project documentation: [../CLAUDE.md](../CLAUDE.md)
- Workload simulator: [../workload-simulator/README.md](../workload-simulator/README.md)
- Database schema scripts: [../scripts/README.md](../scripts/README.md)
