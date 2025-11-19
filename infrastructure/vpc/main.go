package main

import (
	"fmt"

	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/ec2"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		// Load configuration
		cfg := config.New(ctx, "")
		vpcCidr := cfg.Get("vpcCidr")
		if vpcCidr == "" {
			vpcCidr = "10.0.0.0/16"
		}

		projectName := cfg.Get("projectName")
		if projectName == "" {
			projectName = "aurora-bluegreen-lab"
		}

		// Get availability zones
		azs, err := ec2.GetAvailabilityZones(ctx, &ec2.GetAvailabilityZonesArgs{
			State: pulumi.StringRef("available"),
		})
		if err != nil {
			return err
		}

		// Ensure we have at least 2 AZs
		if len(azs.Names) < 2 {
			return fmt.Errorf("need at least 2 availability zones")
		}

		// Create VPC
		vpc, err := ec2.NewVpc(ctx, fmt.Sprintf("%s-vpc", projectName), &ec2.VpcArgs{
			CidrBlock:          pulumi.String(vpcCidr),
			EnableDnsHostnames: pulumi.Bool(true),
			EnableDnsSupport:   pulumi.Bool(true),
			Tags: pulumi.StringMap{
				"Name":    pulumi.String(fmt.Sprintf("%s-vpc", projectName)),
				"Project": pulumi.String(projectName),
			},
		})
		if err != nil {
			return err
		}

		// Create Internet Gateway for public subnet
		igw, err := ec2.NewInternetGateway(ctx, fmt.Sprintf("%s-igw", projectName), &ec2.InternetGatewayArgs{
			VpcId: vpc.ID(),
			Tags: pulumi.StringMap{
				"Name":    pulumi.String(fmt.Sprintf("%s-igw", projectName)),
				"Project": pulumi.String(projectName),
			},
		})
		if err != nil {
			return err
		}

		// Create Aurora Private Subnets (2 AZs)
		auroraSubnet1, err := ec2.NewSubnet(ctx, fmt.Sprintf("%s-aurora-subnet-1", projectName), &ec2.SubnetArgs{
			VpcId:            vpc.ID(),
			CidrBlock:        pulumi.String("10.0.1.0/24"),
			AvailabilityZone: pulumi.String(azs.Names[0]),
			Tags: pulumi.StringMap{
				"Name":    pulumi.String(fmt.Sprintf("%s-aurora-private-subnet-az1", projectName)),
				"Project": pulumi.String(projectName),
				"Type":    pulumi.String("private-aurora"),
			},
		})
		if err != nil {
			return err
		}

		auroraSubnet2, err := ec2.NewSubnet(ctx, fmt.Sprintf("%s-aurora-subnet-2", projectName), &ec2.SubnetArgs{
			VpcId:            vpc.ID(),
			CidrBlock:        pulumi.String("10.0.2.0/24"),
			AvailabilityZone: pulumi.String(azs.Names[1]),
			Tags: pulumi.StringMap{
				"Name":    pulumi.String(fmt.Sprintf("%s-aurora-private-subnet-az2", projectName)),
				"Project": pulumi.String(projectName),
				"Type":    pulumi.String("private-aurora"),
			},
		})
		if err != nil {
			return err
		}

		// Create EC2 Public Subnet (1 AZ)
		ec2Subnet, err := ec2.NewSubnet(ctx, fmt.Sprintf("%s-ec2-subnet", projectName), &ec2.SubnetArgs{
			VpcId:                   vpc.ID(),
			CidrBlock:               pulumi.String("10.0.10.0/24"),
			AvailabilityZone:        pulumi.String(azs.Names[0]),
			MapPublicIpOnLaunch:     pulumi.Bool(true),
			Tags: pulumi.StringMap{
				"Name":    pulumi.String(fmt.Sprintf("%s-ec2-public-subnet-az1", projectName)),
				"Project": pulumi.String(projectName),
				"Type":    pulumi.String("public-ec2"),
			},
		})
		if err != nil {
			return err
		}

		// Create EKS Private Subnets (2 AZs) - Optional
		eksSubnet1, err := ec2.NewSubnet(ctx, fmt.Sprintf("%s-eks-subnet-1", projectName), &ec2.SubnetArgs{
			VpcId:            vpc.ID(),
			CidrBlock:        pulumi.String("10.0.20.0/24"),
			AvailabilityZone: pulumi.String(azs.Names[0]),
			Tags: pulumi.StringMap{
				"Name":    pulumi.String(fmt.Sprintf("%s-eks-private-subnet-az1", projectName)),
				"Project": pulumi.String(projectName),
				"Type":    pulumi.String("private-eks"),
			},
		})
		if err != nil {
			return err
		}

		eksSubnet2, err := ec2.NewSubnet(ctx, fmt.Sprintf("%s-eks-subnet-2", projectName), &ec2.SubnetArgs{
			VpcId:            vpc.ID(),
			CidrBlock:        pulumi.String("10.0.21.0/24"),
			AvailabilityZone: pulumi.String(azs.Names[1]),
			Tags: pulumi.StringMap{
				"Name":    pulumi.String(fmt.Sprintf("%s-eks-private-subnet-az2", projectName)),
				"Project": pulumi.String(projectName),
				"Type":    pulumi.String("private-eks"),
			},
		})
		if err != nil {
			return err
		}

		// Create Route Table for Public Subnet
		publicRouteTable, err := ec2.NewRouteTable(ctx, fmt.Sprintf("%s-public-rt", projectName), &ec2.RouteTableArgs{
			VpcId: vpc.ID(),
			Tags: pulumi.StringMap{
				"Name":    pulumi.String(fmt.Sprintf("%s-public-route-table", projectName)),
				"Project": pulumi.String(projectName),
			},
		})
		if err != nil {
			return err
		}

		// Add route to Internet Gateway
		_, err = ec2.NewRoute(ctx, fmt.Sprintf("%s-public-route", projectName), &ec2.RouteArgs{
			RouteTableId:         publicRouteTable.ID(),
			DestinationCidrBlock: pulumi.String("0.0.0.0/0"),
			GatewayId:            igw.ID(),
		})
		if err != nil {
			return err
		}

		// Associate public route table with EC2 subnet
		_, err = ec2.NewRouteTableAssociation(ctx, fmt.Sprintf("%s-ec2-rt-assoc", projectName), &ec2.RouteTableAssociationArgs{
			SubnetId:     ec2Subnet.ID(),
			RouteTableId: publicRouteTable.ID(),
		})
		if err != nil {
			return err
		}

		// Create Route Table for Private Subnets (Aurora and EKS)
		privateRouteTable, err := ec2.NewRouteTable(ctx, fmt.Sprintf("%s-private-rt", projectName), &ec2.RouteTableArgs{
			VpcId: vpc.ID(),
			Tags: pulumi.StringMap{
				"Name":    pulumi.String(fmt.Sprintf("%s-private-route-table", projectName)),
				"Project": pulumi.String(projectName),
			},
		})
		if err != nil {
			return err
		}

		// Associate private route table with Aurora subnets
		_, err = ec2.NewRouteTableAssociation(ctx, fmt.Sprintf("%s-aurora-rt-assoc-1", projectName), &ec2.RouteTableAssociationArgs{
			SubnetId:     auroraSubnet1.ID(),
			RouteTableId: privateRouteTable.ID(),
		})
		if err != nil {
			return err
		}

		_, err = ec2.NewRouteTableAssociation(ctx, fmt.Sprintf("%s-aurora-rt-assoc-2", projectName), &ec2.RouteTableAssociationArgs{
			SubnetId:     auroraSubnet2.ID(),
			RouteTableId: privateRouteTable.ID(),
		})
		if err != nil {
			return err
		}

		// Associate private route table with EKS subnets
		_, err = ec2.NewRouteTableAssociation(ctx, fmt.Sprintf("%s-eks-rt-assoc-1", projectName), &ec2.RouteTableAssociationArgs{
			SubnetId:     eksSubnet1.ID(),
			RouteTableId: privateRouteTable.ID(),
		})
		if err != nil {
			return err
		}

		_, err = ec2.NewRouteTableAssociation(ctx, fmt.Sprintf("%s-eks-rt-assoc-2", projectName), &ec2.RouteTableAssociationArgs{
			SubnetId:     eksSubnet2.ID(),
			RouteTableId: privateRouteTable.ID(),
		})
		if err != nil {
			return err
		}

		// Create Security Group for Aurora
		auroraSg, err := ec2.NewSecurityGroup(ctx, fmt.Sprintf("%s-aurora-sg", projectName), &ec2.SecurityGroupArgs{
			VpcId:       vpc.ID(),
			Description: pulumi.String("Security group for Aurora MySQL cluster"),
			Ingress: ec2.SecurityGroupIngressArray{
				&ec2.SecurityGroupIngressArgs{
					Protocol:   pulumi.String("tcp"),
					FromPort:   pulumi.Int(3306),
					ToPort:     pulumi.Int(3306),
					CidrBlocks: pulumi.StringArray{
						pulumi.String("10.0.10.0/24"), // EC2 subnet
						pulumi.String("10.0.20.0/24"), // EKS subnet 1
						pulumi.String("10.0.21.0/24"), // EKS subnet 2
					},
					Description: pulumi.String("MySQL access from EC2 and EKS subnets"),
				},
			},
			Egress: ec2.SecurityGroupEgressArray{
				&ec2.SecurityGroupEgressArgs{
					Protocol:   pulumi.String("-1"),
					FromPort:   pulumi.Int(0),
					ToPort:     pulumi.Int(0),
					CidrBlocks: pulumi.StringArray{pulumi.String("0.0.0.0/0")},
				},
			},
			Tags: pulumi.StringMap{
				"Name":    pulumi.String(fmt.Sprintf("%s-aurora-sg", projectName)),
				"Project": pulumi.String(projectName),
			},
		})
		if err != nil {
			return err
		}

		// Create Security Group for EC2
		ec2Sg, err := ec2.NewSecurityGroup(ctx, fmt.Sprintf("%s-ec2-sg", projectName), &ec2.SecurityGroupArgs{
			VpcId:       vpc.ID(),
			Description: pulumi.String("Security group for EC2 workload simulator"),
			Ingress: ec2.SecurityGroupIngressArray{
				&ec2.SecurityGroupIngressArgs{
					Protocol:    pulumi.String("tcp"),
					FromPort:    pulumi.Int(22),
					ToPort:      pulumi.Int(22),
					CidrBlocks:  pulumi.StringArray{pulumi.String("0.0.0.0/0")},
					Description: pulumi.String("SSH access"),
				},
			},
			Egress: ec2.SecurityGroupEgressArray{
				&ec2.SecurityGroupEgressArgs{
					Protocol:   pulumi.String("-1"),
					FromPort:   pulumi.Int(0),
					ToPort:     pulumi.Int(0),
					CidrBlocks: pulumi.StringArray{pulumi.String("0.0.0.0/0")},
				},
			},
			Tags: pulumi.StringMap{
				"Name":    pulumi.String(fmt.Sprintf("%s-ec2-sg", projectName)),
				"Project": pulumi.String(projectName),
			},
		})
		if err != nil {
			return err
		}

		// Create Security Group for EKS
		eksSg, err := ec2.NewSecurityGroup(ctx, fmt.Sprintf("%s-eks-sg", projectName), &ec2.SecurityGroupArgs{
			VpcId:       vpc.ID(),
			Description: pulumi.String("Security group for EKS cluster nodes"),
			Egress: ec2.SecurityGroupEgressArray{
				&ec2.SecurityGroupEgressArgs{
					Protocol:   pulumi.String("-1"),
					FromPort:   pulumi.Int(0),
					ToPort:     pulumi.Int(0),
					CidrBlocks: pulumi.StringArray{pulumi.String("0.0.0.0/0")},
				},
			},
			Tags: pulumi.StringMap{
				"Name":    pulumi.String(fmt.Sprintf("%s-eks-sg", projectName)),
				"Project": pulumi.String(projectName),
			},
		})
		if err != nil {
			return err
		}

		// Allow EKS nodes to communicate with each other
		_, err = ec2.NewSecurityGroupRule(ctx, fmt.Sprintf("%s-eks-self-ingress", projectName), &ec2.SecurityGroupRuleArgs{
			Type:                  pulumi.String("ingress"),
			FromPort:              pulumi.Int(0),
			ToPort:                pulumi.Int(65535),
			Protocol:              pulumi.String("-1"),
			SourceSecurityGroupId: eksSg.ID(),
			SecurityGroupId:       eksSg.ID(),
			Description:           pulumi.String("Allow nodes to communicate with each other"),
		})
		if err != nil {
			return err
		}

		// Export outputs
		ctx.Export("vpcId", vpc.ID())
		ctx.Export("vpcCidr", vpc.CidrBlock)
		ctx.Export("auroraSubnet1Id", auroraSubnet1.ID())
		ctx.Export("auroraSubnet2Id", auroraSubnet2.ID())
		ctx.Export("ec2SubnetId", ec2Subnet.ID())
		ctx.Export("eksSubnet1Id", eksSubnet1.ID())
		ctx.Export("eksSubnet2Id", eksSubnet2.ID())
		ctx.Export("auroraSecurityGroupId", auroraSg.ID())
		ctx.Export("ec2SecurityGroupId", ec2Sg.ID())
		ctx.Export("eksSecurityGroupId", eksSg.ID())
		ctx.Export("internetGatewayId", igw.ID())
		ctx.Export("publicRouteTableId", publicRouteTable.ID())
		ctx.Export("privateRouteTableId", privateRouteTable.ID())
		ctx.Export("availabilityZone1", pulumi.String(azs.Names[0]))
		ctx.Export("availabilityZone2", pulumi.String(azs.Names[1]))

		return nil
	})
}
