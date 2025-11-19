#!/bin/bash

# Aurora Blue-Green Deployment Lab - Infrastructure Deployment Script
# This script automates the deployment of VPC, Aurora, and EC2 infrastructure

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Check prerequisites
print_info "Checking prerequisites..."

if ! command_exists pulumi; then
    print_error "Pulumi CLI is not installed. Please install from https://www.pulumi.com/docs/get-started/install/"
    exit 1
fi

if ! command_exists go; then
    print_error "Go is not installed. Please install from https://go.dev/doc/install"
    exit 1
fi

if ! command_exists aws; then
    print_error "AWS CLI is not installed. Please install from https://aws.amazon.com/cli/"
    exit 1
fi

# Check AWS credentials
if ! aws sts get-caller-identity >/dev/null 2>&1; then
    print_error "AWS credentials are not configured. Please run 'aws configure'"
    exit 1
fi

print_success "All prerequisites are installed"

# Get configuration from user
print_info "Please provide the following configuration:"

read -p "AWS Region (default: us-east-1): " AWS_REGION
AWS_REGION=${AWS_REGION:-us-east-1}

read -p "Project Name (default: aurora-bluegreen-lab): " PROJECT_NAME
PROJECT_NAME=${PROJECT_NAME:-aurora-bluegreen-lab}

read -p "Stack Name (default: dev): " STACK_NAME
STACK_NAME=${STACK_NAME:-dev}

read -p "VPC CIDR (default: 10.0.0.0/16): " VPC_CIDR
VPC_CIDR=${VPC_CIDR:-10.0.0.0/16}

read -sp "Aurora Master Password (min 8 characters): " MASTER_PASSWORD
echo
if [ ${#MASTER_PASSWORD} -lt 8 ]; then
    print_error "Password must be at least 8 characters"
    exit 1
fi

read -p "EC2 Key Pair Name: " KEY_NAME
if [ -z "$KEY_NAME" ]; then
    print_error "EC2 Key Pair Name is required"
    exit 1
fi

# Verify key pair exists
if ! aws ec2 describe-key-pairs --key-names "$KEY_NAME" --region "$AWS_REGION" >/dev/null 2>&1; then
    print_error "Key pair '$KEY_NAME' does not exist in region $AWS_REGION"
    print_info "Create it with: aws ec2 create-key-pair --key-name $KEY_NAME --query 'KeyMaterial' --output text > ${KEY_NAME}.pem"
    exit 1
fi

print_success "Configuration collected"

# Get Pulumi organization
PULUMI_ORG=$(pulumi whoami 2>/dev/null || echo "")
if [ -z "$PULUMI_ORG" ]; then
    print_error "Not logged in to Pulumi. Please run 'pulumi login'"
    exit 1
fi

print_info "Pulumi Organization: $PULUMI_ORG"

# Construct stack references
VPC_STACK_REF="${PULUMI_ORG}/aurora-bluegreen-vpc/${STACK_NAME}"
AURORA_STACK_REF="${PULUMI_ORG}/aurora-bluegreen-aurora/${STACK_NAME}"

echo ""
print_info "=== Deployment Configuration ==="
echo "AWS Region: $AWS_REGION"
echo "Project Name: $PROJECT_NAME"
echo "Stack Name: $STACK_NAME"
echo "VPC CIDR: $VPC_CIDR"
echo "EC2 Key Pair: $KEY_NAME"
echo "VPC Stack Reference: $VPC_STACK_REF"
echo "Aurora Stack Reference: $AURORA_STACK_REF"
echo ""

read -p "Proceed with deployment? (yes/no): " CONFIRM
if [ "$CONFIRM" != "yes" ]; then
    print_warning "Deployment cancelled"
    exit 0
fi

# Step 1: Deploy VPC
print_info "=========================================="
print_info "Step 1: Deploying VPC Infrastructure"
print_info "=========================================="

cd vpc

# Initialize stack if it doesn't exist
if ! pulumi stack select "$STACK_NAME" 2>/dev/null; then
    print_info "Creating new stack: $STACK_NAME"
    pulumi stack init "$STACK_NAME"
fi

# Configure stack
print_info "Configuring VPC stack..."
pulumi config set aws:region "$AWS_REGION"
pulumi config set vpcCidr "$VPC_CIDR"
pulumi config set projectName "$PROJECT_NAME"

# Deploy
print_info "Deploying VPC infrastructure..."
pulumi up --yes

VPC_ID=$(pulumi stack output vpcId)
print_success "VPC deployed successfully: $VPC_ID"

cd ..

# Step 2: Deploy Aurora
print_info "=========================================="
print_info "Step 2: Deploying Aurora Cluster"
print_info "=========================================="
print_warning "This step takes approximately 10-15 minutes"

cd aurora

# Initialize stack if it doesn't exist
if ! pulumi stack select "$STACK_NAME" 2>/dev/null; then
    print_info "Creating new stack: $STACK_NAME"
    pulumi stack init "$STACK_NAME"
fi

# Configure stack
print_info "Configuring Aurora stack..."
pulumi config set aws:region "$AWS_REGION"
pulumi config set vpcStackName "$VPC_STACK_REF"
pulumi config set projectName "$PROJECT_NAME"
pulumi config set databaseName "lab_db"
pulumi config set masterUsername "admin"
pulumi config set --secret masterPassword "$MASTER_PASSWORD"
pulumi config set engineVersion "8.0.mysql_aurora.3.04.0"
pulumi config set instanceClass "db.r6g.xlarge"

# Deploy
print_info "Deploying Aurora cluster..."
pulumi up --yes

AURORA_ENDPOINT=$(pulumi stack output clusterEndpoint)
print_success "Aurora cluster deployed successfully: $AURORA_ENDPOINT"

cd ..

# Step 3: Initialize Schema (prompt user)
print_info "=========================================="
print_info "Step 3: Database Schema Initialization"
print_info "=========================================="
print_warning "The schema initialization creates 12,000 tables and takes 30-60 minutes"
print_info "You can run this later with: cd scripts && ./init-schema.sh"
echo ""

read -p "Initialize database schema now? (yes/no): " INIT_SCHEMA
if [ "$INIT_SCHEMA" == "yes" ]; then
    if [ -f "../scripts/init-schema.sh" ]; then
        print_info "Starting schema initialization..."
        cd ../scripts
        ./init-schema.sh
        cd ../infrastructure
        print_success "Schema initialization completed"
    else
        print_warning "Schema initialization script not found at ../scripts/init-schema.sh"
    fi
else
    print_warning "Skipping schema initialization. Run it manually before testing."
fi

# Step 4: Deploy EC2
print_info "=========================================="
print_info "Step 4: Deploying EC2 Workload Simulator"
print_info "=========================================="

cd ec2

# Initialize stack if it doesn't exist
if ! pulumi stack select "$STACK_NAME" 2>/dev/null; then
    print_info "Creating new stack: $STACK_NAME"
    pulumi stack init "$STACK_NAME"
fi

# Configure stack
print_info "Configuring EC2 stack..."
pulumi config set aws:region "$AWS_REGION"
pulumi config set vpcStackName "$VPC_STACK_REF"
pulumi config set auroraStackName "$AURORA_STACK_REF"
pulumi config set projectName "$PROJECT_NAME"
pulumi config set keyName "$KEY_NAME"
pulumi config set instanceType "t3.xlarge"

# Deploy
print_info "Deploying EC2 instance..."
pulumi up --yes

EC2_PUBLIC_IP=$(pulumi stack output publicIp)
SSH_COMMAND=$(pulumi stack output sshCommand)
print_success "EC2 instance deployed successfully: $EC2_PUBLIC_IP"

cd ..

# Deployment Summary
print_info "=========================================="
print_success "Deployment Completed Successfully!"
print_info "=========================================="
echo ""
print_info "Infrastructure Details:"
echo "  VPC ID: $VPC_ID"
echo "  Aurora Endpoint: $AURORA_ENDPOINT"
echo "  EC2 Public IP: $EC2_PUBLIC_IP"
echo ""
print_info "Next Steps:"
echo ""
echo "1. Build the workload simulator (if not already done):"
echo "   cd ../workload-simulator"
echo "   mvn clean package"
echo ""
echo "2. Upload the workload simulator to EC2:"
echo "   scp -i ${KEY_NAME}.pem target/workload-simulator.jar ec2-user@${EC2_PUBLIC_IP}:/opt/workload-simulator/"
echo ""
echo "3. SSH into the EC2 instance:"
echo "   $SSH_COMMAND"
echo ""
echo "4. Run the workload simulator:"
echo "   cd /opt/workload-simulator"
echo "   ./run-simulator.sh $AURORA_ENDPOINT"
echo ""
echo "5. Test the Blue-Green deployment:"
echo "   aws rds create-blue-green-deployment \\"
echo "     --blue-green-deployment-name aurora-upgrade-test \\"
echo "     --source-arn <aurora-cluster-arn> \\"
echo "     --target-engine-version 8.0.mysql_aurora.3.10.0"
echo ""
print_info "For detailed instructions, see the README files in each infrastructure directory."
echo ""
print_success "Happy testing! 🚀"
