package main

import (
	"encoding/base64"
	"fmt"

	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/ec2"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		// Load configuration
		cfg := config.New(ctx, "")

		projectName := cfg.Get("projectName")
		if projectName == "" {
			projectName = "aurora-bluegreen-lab"
		}

		instanceType := cfg.Get("instanceType")
		if instanceType == "" {
			instanceType = "t3.xlarge"
		}

		keyName := cfg.Get("keyName")
		if keyName == "" {
			return fmt.Errorf("keyName is required. Please set it with: pulumi config set keyName <your-key-pair-name>")
		}

		// Reference VPC stack outputs
		vpcStack := cfg.Require("vpcStackName")
		vpcStackRef, err := pulumi.NewStackReference(ctx, vpcStack, nil)
		if err != nil {
			return err
		}

		ec2SubnetId := vpcStackRef.GetStringOutput(pulumi.String("ec2SubnetId"))
		ec2SecurityGroupId := vpcStackRef.GetStringOutput(pulumi.String("ec2SecurityGroupId"))

		// Reference Aurora stack outputs (optional, for convenience)
		auroraStackName := cfg.Get("auroraStackName")
		var clusterEndpoint pulumi.StringOutput
		if auroraStackName != "" {
			auroraStackRef, err := pulumi.NewStackReference(ctx, auroraStackName, nil)
			if err == nil {
				clusterEndpoint = auroraStackRef.GetStringOutput(pulumi.String("clusterEndpoint"))
			}
		}

		// Get the latest Amazon Linux 2023 AMI
		ami, err := ec2.LookupAmi(ctx, &ec2.LookupAmiArgs{
			MostRecent: pulumi.BoolRef(true),
			Owners:     []string{"amazon"},
			Filters: []ec2.GetAmiFilter{
				{
					Name:   "name",
					Values: []string{"al2023-ami-2023.*-x86_64"},
				},
				{
					Name:   "architecture",
					Values: []string{"x86_64"},
				},
				{
					Name:   "virtualization-type",
					Values: []string{"hvm"},
				},
			},
		})
		if err != nil {
			return err
		}

		// User data script to install Java and prepare the workload simulator
		userData := `#!/bin/bash
set -e

# Update system
yum update -y

# Install Amazon Corretto 17 (OpenJDK)
yum install -y java-17-amazon-corretto-headless

# Install MySQL client for testing
yum install -y mysql

# Install git (for cloning the workload simulator if needed)
yum install -y git

# Create directory for workload simulator
mkdir -p /opt/workload-simulator
chown ec2-user:ec2-user /opt/workload-simulator

# Create a helper script to run the workload simulator
cat > /opt/workload-simulator/run-simulator.sh << 'EOF'
#!/bin/bash
# Helper script to run the workload simulator
# Usage: ./run-simulator.sh <aurora-endpoint> [additional-options]

if [ -z "$1" ]; then
  echo "Usage: $0 <aurora-endpoint> [additional-options]"
  echo "Example: $0 my-cluster.cluster-xxxxx.us-east-1.rds.amazonaws.com --write-workers 10"
  exit 1
fi

AURORA_ENDPOINT=$1
shift

java -jar /opt/workload-simulator/workload-simulator.jar \
  --aurora-endpoint "$AURORA_ENDPOINT" \
  --database-name lab_db \
  --write-workers 10 \
  --write-rate 100 \
  --connection-pool-size 100 \
  "$@"
EOF

chmod +x /opt/workload-simulator/run-simulator.sh
chown ec2-user:ec2-user /opt/workload-simulator/run-simulator.sh

# Create a README with instructions
cat > /opt/workload-simulator/README.txt << 'EOF'
Aurora Blue-Green Deployment Lab - Workload Simulator

This directory contains the workload simulator for testing Aurora Blue-Green deployments.

SETUP:
1. Upload the workload-simulator.jar file to this directory:
   scp -i your-key.pem workload-simulator.jar ec2-user@<instance-ip>:/opt/workload-simulator/

USAGE:
1. Run the workload simulator directly:
   java -jar workload-simulator.jar \
     --aurora-endpoint <your-cluster-endpoint> \
     --database-name lab_db \
     --write-workers 10 \
     --write-rate 100 \
     --connection-pool-size 100

2. Or use the helper script:
   ./run-simulator.sh <your-cluster-endpoint>

3. To run with custom parameters:
   ./run-simulator.sh <your-cluster-endpoint> --write-workers 20 --write-rate 200

AVAILABLE PARAMETERS:
  --aurora-endpoint       : Aurora cluster writer endpoint (required)
  --database-name         : Database name (default: lab_db)
  --write-workers         : Number of concurrent write workers (default: 10)
  --write-rate            : Writes per second per worker (default: 100)
  --connection-pool-size  : Database connection pool size (default: 100)
  --log-interval          : Statistics log interval in seconds (default: 10)

TESTING THE BLUE-GREEN DEPLOYMENT:
1. Start the workload simulator
2. Observe the console output showing successful write operations
3. In AWS Console or CLI, create a Blue-Green deployment for your Aurora cluster
4. Keep the workload simulator running during the upgrade
5. Watch for connection errors during the Blue-Green switchover
6. Validate that the workload resumes after the switchover completes

For more information, see the project documentation at:
/home/ec2-user/aurora-blue-green-deployment-lab/README.md
EOF

chown ec2-user:ec2-user /opt/workload-simulator/README.txt

echo "EC2 instance setup completed successfully" > /var/log/user-data.log
`

		userDataEncoded := pulumi.String(userData).ToStringOutput().ApplyT(func(s string) string {
			return base64.StdEncoding.EncodeToString([]byte(s))
		}).(pulumi.StringOutput)

		// Create EC2 instance
		instance, err := ec2.NewInstance(ctx, fmt.Sprintf("%s-workload-simulator", projectName), &ec2.InstanceArgs{
			InstanceType:               pulumi.String(instanceType),
			Ami:                        pulumi.String(ami.Id),
			SubnetId:                   ec2SubnetId,
			VpcSecurityGroupIds:        pulumi.StringArray{ec2SecurityGroupId},
			KeyName:                    pulumi.String(keyName),
			UserDataBase64:             userDataEncoded,
			AssociatePublicIpAddress:   pulumi.Bool(true),
			DisableApiTermination:      pulumi.Bool(false),
			InstanceInitiatedShutdownBehavior: pulumi.String("stop"),
			Monitoring:                 pulumi.Bool(true),
			EbsOptimized:               pulumi.Bool(true),
			RootBlockDevice: &ec2.InstanceRootBlockDeviceArgs{
				VolumeSize:          pulumi.Int(30),
				VolumeType:          pulumi.String("gp3"),
				DeleteOnTermination: pulumi.Bool(true),
				Encrypted:           pulumi.Bool(true),
			},
			Tags: pulumi.StringMap{
				"Name":    pulumi.String(fmt.Sprintf("%s-workload-simulator", projectName)),
				"Project": pulumi.String(projectName),
				"Role":    pulumi.String("workload-simulator"),
			},
		})
		if err != nil {
			return err
		}

		// Export outputs
		ctx.Export("instanceId", instance.ID())
		ctx.Export("publicIp", instance.PublicIp)
		ctx.Export("publicDns", instance.PublicDns)
		ctx.Export("privateIp", instance.PrivateIp)
		ctx.Export("instanceType", instance.InstanceType)
		ctx.Export("availabilityZone", instance.AvailabilityZone)

		// Export connection information
		ctx.Export("sshCommand", pulumi.Sprintf("ssh -i %s.pem ec2-user@%s", keyName, instance.PublicDns))
		ctx.Export("workloadSimulatorPath", pulumi.String("/opt/workload-simulator"))

		// Export Aurora endpoint if available
		if auroraStackName != "" && clusterEndpoint != nil {
			ctx.Export("auroraClusterEndpoint", clusterEndpoint)
			ctx.Export("runSimulatorCommand", pulumi.Sprintf(
				"/opt/workload-simulator/run-simulator.sh %s",
				clusterEndpoint,
			))
		}

		return nil
	})
}
