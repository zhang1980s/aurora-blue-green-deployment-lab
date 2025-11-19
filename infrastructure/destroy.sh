#!/bin/bash

# Aurora Blue-Green Deployment Lab - Infrastructure Destroy Script
# This script safely destroys the infrastructure in the correct order

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

# Get stack name
read -p "Stack Name to destroy (default: dev): " STACK_NAME
STACK_NAME=${STACK_NAME:-dev}

print_warning "=========================================="
print_warning "WARNING: This will destroy all infrastructure!"
print_warning "=========================================="
echo ""
print_info "This script will destroy the following stacks:"
echo "  1. EC2 Workload Simulator"
echo "  2. Aurora MySQL Cluster"
echo "  3. VPC and Network Infrastructure"
echo ""
print_warning "This action cannot be undone!"
print_warning "All data in the Aurora cluster will be permanently deleted!"
echo ""

read -p "Are you sure you want to proceed? (type 'yes' to confirm): " CONFIRM
if [ "$CONFIRM" != "yes" ]; then
    print_info "Destroy cancelled"
    exit 0
fi

# Step 1: Destroy EC2
print_info "=========================================="
print_info "Step 1: Destroying EC2 Workload Simulator"
print_info "=========================================="

if [ -d "ec2" ]; then
    cd ec2
    if pulumi stack select "$STACK_NAME" 2>/dev/null; then
        print_info "Destroying EC2 stack..."
        pulumi destroy --yes
        print_success "EC2 stack destroyed"

        # Optionally remove the stack
        read -p "Remove the EC2 Pulumi stack? (yes/no): " REMOVE_STACK
        if [ "$REMOVE_STACK" == "yes" ]; then
            pulumi stack rm "$STACK_NAME" --yes
            print_success "EC2 stack removed"
        fi
    else
        print_warning "EC2 stack '$STACK_NAME' not found, skipping"
    fi
    cd ..
else
    print_warning "EC2 directory not found, skipping"
fi

# Step 2: Destroy Aurora
print_info "=========================================="
print_info "Step 2: Destroying Aurora Cluster"
print_info "=========================================="
print_warning "This will permanently delete your Aurora cluster and all data!"

read -p "Proceed with Aurora cluster destruction? (yes/no): " CONFIRM_AURORA
if [ "$CONFIRM_AURORA" != "yes" ]; then
    print_warning "Aurora destruction cancelled. Please destroy manually if needed."
    print_info "To destroy Aurora manually: cd aurora && pulumi destroy"
else
    if [ -d "aurora" ]; then
        cd aurora
        if pulumi stack select "$STACK_NAME" 2>/dev/null; then
            print_info "Destroying Aurora stack..."
            pulumi destroy --yes
            print_success "Aurora stack destroyed"

            # Optionally remove the stack
            read -p "Remove the Aurora Pulumi stack? (yes/no): " REMOVE_STACK
            if [ "$REMOVE_STACK" == "yes" ]; then
                pulumi stack rm "$STACK_NAME" --yes
                print_success "Aurora stack removed"
            fi
        else
            print_warning "Aurora stack '$STACK_NAME' not found, skipping"
        fi
        cd ..
    else
        print_warning "Aurora directory not found, skipping"
    fi
fi

# Step 3: Destroy VPC
print_info "=========================================="
print_info "Step 3: Destroying VPC Infrastructure"
print_info "=========================================="

if [ -d "vpc" ]; then
    cd vpc
    if pulumi stack select "$STACK_NAME" 2>/dev/null; then
        print_info "Destroying VPC stack..."
        pulumi destroy --yes
        print_success "VPC stack destroyed"

        # Optionally remove the stack
        read -p "Remove the VPC Pulumi stack? (yes/no): " REMOVE_STACK
        if [ "$REMOVE_STACK" == "yes" ]; then
            pulumi stack rm "$STACK_NAME" --yes
            print_success "VPC stack removed"
        fi
    else
        print_warning "VPC stack '$STACK_NAME' not found, skipping"
    fi
    cd ..
else
    print_warning "VPC directory not found, skipping"
fi

# Summary
print_info "=========================================="
print_success "Destruction Completed!"
print_info "=========================================="
echo ""
print_info "All infrastructure has been destroyed successfully."
print_info "You can redeploy at any time using ./deploy.sh"
echo ""
print_warning "Note: Pulumi state files are preserved unless you explicitly removed the stacks."
print_info "To completely clean up Pulumi state, run in each directory:"
echo "  pulumi stack rm $STACK_NAME --yes"
echo ""
