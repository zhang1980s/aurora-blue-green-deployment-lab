.PHONY: help deploy destroy vpc aurora ec2 clean outputs test-connection

# Default target
help:
	@echo "Aurora Blue-Green Deployment Lab - Infrastructure Commands"
	@echo ""
	@echo "Available targets:"
	@echo "  make deploy          - Deploy all infrastructure (VPC -> Aurora -> EC2)"
	@echo "  make destroy         - Destroy all infrastructure in reverse order"
	@echo "  make vpc             - Deploy only VPC infrastructure"
	@echo "  make aurora          - Deploy only Aurora cluster (requires VPC)"
	@echo "  make ec2             - Deploy only EC2 instance (requires VPC)"
	@echo "  make outputs         - Show all stack outputs"
	@echo "  make clean           - Clean up Go modules and build artifacts"
	@echo "  make test-connection - Test Aurora connection from EC2"
	@echo ""
	@echo "Environment variables:"
	@echo "  STACK_NAME          - Pulumi stack name (default: dev)"
	@echo "  AWS_REGION          - AWS region (default: us-east-1)"
	@echo ""
	@echo "Examples:"
	@echo "  make deploy STACK_NAME=prod AWS_REGION=us-west-2"
	@echo "  make outputs STACK_NAME=dev"

# Variables
STACK_NAME ?= dev
AWS_REGION ?= us-east-1

# Deploy all infrastructure
deploy:
	@echo "Deploying all infrastructure..."
	@./deploy.sh

# Destroy all infrastructure
destroy:
	@echo "Destroying all infrastructure..."
	@./destroy.sh

# Deploy VPC only
vpc:
	@echo "Deploying VPC infrastructure..."
	cd vpc && \
	pulumi stack select $(STACK_NAME) || pulumi stack init $(STACK_NAME) && \
	pulumi config set aws:region $(AWS_REGION) && \
	pulumi up

# Deploy Aurora only
aurora:
	@echo "Deploying Aurora cluster..."
	@if [ -z "$(VPC_STACK)" ]; then \
		echo "Error: VPC_STACK environment variable is required"; \
		echo "Example: make aurora VPC_STACK=myorg/aurora-bluegreen-vpc/dev"; \
		exit 1; \
	fi
	cd aurora && \
	pulumi stack select $(STACK_NAME) || pulumi stack init $(STACK_NAME) && \
	pulumi config set aws:region $(AWS_REGION) && \
	pulumi config set vpcStackName $(VPC_STACK) && \
	pulumi up

# Deploy EC2 only
ec2:
	@echo "Deploying EC2 instance..."
	@if [ -z "$(VPC_STACK)" ]; then \
		echo "Error: VPC_STACK environment variable is required"; \
		echo "Example: make ec2 VPC_STACK=myorg/aurora-bluegreen-vpc/dev KEY_NAME=my-key"; \
		exit 1; \
	fi
	@if [ -z "$(KEY_NAME)" ]; then \
		echo "Error: KEY_NAME environment variable is required"; \
		echo "Example: make ec2 VPC_STACK=myorg/aurora-bluegreen-vpc/dev KEY_NAME=my-key"; \
		exit 1; \
	fi
	cd ec2 && \
	pulumi stack select $(STACK_NAME) || pulumi stack init $(STACK_NAME) && \
	pulumi config set aws:region $(AWS_REGION) && \
	pulumi config set vpcStackName $(VPC_STACK) && \
	pulumi config set keyName $(KEY_NAME) && \
	pulumi up

# Show all outputs
outputs:
	@echo "=== VPC Outputs ==="
	@cd vpc && pulumi stack select $(STACK_NAME) 2>/dev/null && pulumi stack output --json || echo "Stack not found"
	@echo ""
	@echo "=== Aurora Outputs ==="
	@cd aurora && pulumi stack select $(STACK_NAME) 2>/dev/null && pulumi stack output --json || echo "Stack not found"
	@echo ""
	@echo "=== EC2 Outputs ==="
	@cd ec2 && pulumi stack select $(STACK_NAME) 2>/dev/null && pulumi stack output --json || echo "Stack not found"

# Clean Go modules and build artifacts
clean:
	@echo "Cleaning Go modules and build artifacts..."
	cd vpc && rm -f go.sum && go mod tidy
	cd aurora && rm -f go.sum && go mod tidy
	cd ec2 && rm -f go.sum && go mod tidy
	@echo "Clean complete"

# Test Aurora connection
test-connection:
	@echo "Testing Aurora connection from EC2..."
	@AURORA_ENDPOINT=$$(cd aurora && pulumi stack select $(STACK_NAME) 2>/dev/null && pulumi stack output clusterEndpoint) && \
	EC2_IP=$$(cd ec2 && pulumi stack select $(STACK_NAME) 2>/dev/null && pulumi stack output publicIp) && \
	if [ -z "$$AURORA_ENDPOINT" ] || [ -z "$$EC2_IP" ]; then \
		echo "Error: Aurora or EC2 stack outputs not found"; \
		exit 1; \
	fi && \
	echo "Aurora Endpoint: $$AURORA_ENDPOINT" && \
	echo "EC2 IP: $$EC2_IP" && \
	echo "Testing connection..." && \
	echo "Note: You need to have SSH key configured to test connection"

# Preview changes for all stacks
preview:
	@echo "=== VPC Preview ==="
	@cd vpc && pulumi stack select $(STACK_NAME) 2>/dev/null && pulumi preview || echo "Stack not found"
	@echo ""
	@echo "=== Aurora Preview ==="
	@cd aurora && pulumi stack select $(STACK_NAME) 2>/dev/null && pulumi preview || echo "Stack not found"
	@echo ""
	@echo "=== EC2 Preview ==="
	@cd ec2 && pulumi stack select $(STACK_NAME) 2>/dev/null && pulumi preview || echo "Stack not found"

# Refresh all stacks
refresh:
	@echo "Refreshing all stacks..."
	@cd vpc && pulumi stack select $(STACK_NAME) 2>/dev/null && pulumi refresh --yes || echo "VPC stack not found"
	@cd aurora && pulumi stack select $(STACK_NAME) 2>/dev/null && pulumi refresh --yes || echo "Aurora stack not found"
	@cd ec2 && pulumi stack select $(STACK_NAME) 2>/dev/null && pulumi refresh --yes || echo "EC2 stack not found"
