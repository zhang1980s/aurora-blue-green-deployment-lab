package main

import (
	"fmt"

	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/rds"
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

		dbName := cfg.Get("databaseName")
		if dbName == "" {
			dbName = "lab_db"
		}

		dbUsername := cfg.Get("masterUsername")
		if dbUsername == "" {
			dbUsername = "admin"
		}

		dbPassword := cfg.RequireSecret("masterPassword")

		engineVersion := cfg.Get("engineVersion")
		if engineVersion == "" {
			engineVersion = "8.0.mysql_aurora.3.04.0"
		}

		instanceClass := cfg.Get("instanceClass")
		if instanceClass == "" {
			instanceClass = "db.r6g.xlarge"
		}

		// Reference VPC stack outputs
		vpcStack := cfg.Require("vpcStackName")
		vpcStackRef, err := pulumi.NewStackReference(ctx, vpcStack, nil)
		if err != nil {
			return err
		}

		auroraSubnet1Id := vpcStackRef.GetStringOutput(pulumi.String("auroraSubnet1Id"))
		auroraSubnet2Id := vpcStackRef.GetStringOutput(pulumi.String("auroraSubnet2Id"))
		auroraSecurityGroupId := vpcStackRef.GetStringOutput(pulumi.String("auroraSecurityGroupId"))

		// Create DB Subnet Group
		dbSubnetGroup, err := rds.NewSubnetGroup(ctx, fmt.Sprintf("%s-db-subnet-group", projectName), &rds.SubnetGroupArgs{
			Name: pulumi.String(fmt.Sprintf("%s-aurora-subnet-group", projectName)),
			SubnetIds: pulumi.StringArray{
				auroraSubnet1Id,
				auroraSubnet2Id,
			},
			Tags: pulumi.StringMap{
				"Name":    pulumi.String(fmt.Sprintf("%s-aurora-subnet-group", projectName)),
				"Project": pulumi.String(projectName),
			},
		})
		if err != nil {
			return err
		}

		// Create DB Cluster Parameter Group
		clusterParameterGroup, err := rds.NewClusterParameterGroup(ctx, fmt.Sprintf("%s-cluster-pg", projectName), &rds.ClusterParameterGroupArgs{
			Name:        pulumi.String(fmt.Sprintf("%s-aurora-cluster-pg", projectName)),
			Family:      pulumi.String("aurora-mysql8.0"),
			Description: pulumi.String("Cluster parameter group for Aurora Blue-Green lab"),
			Parameters: rds.ClusterParameterGroupParameterArray{
				&rds.ClusterParameterGroupParameterArgs{
					Name:  pulumi.String("character_set_server"),
					Value: pulumi.String("utf8mb4"),
				},
				&rds.ClusterParameterGroupParameterArgs{
					Name:  pulumi.String("collation_server"),
					Value: pulumi.String("utf8mb4_unicode_ci"),
				},
			},
			Tags: pulumi.StringMap{
				"Name":    pulumi.String(fmt.Sprintf("%s-aurora-cluster-pg", projectName)),
				"Project": pulumi.String(projectName),
			},
		})
		if err != nil {
			return err
		}

		// Create DB Parameter Group (for instances)
		instanceParameterGroup, err := rds.NewParameterGroup(ctx, fmt.Sprintf("%s-instance-pg", projectName), &rds.ParameterGroupArgs{
			Name:        pulumi.String(fmt.Sprintf("%s-aurora-instance-pg", projectName)),
			Family:      pulumi.String("aurora-mysql8.0"),
			Description: pulumi.String("Instance parameter group for Aurora Blue-Green lab"),
			Parameters: rds.ParameterGroupParameterArray{
				&rds.ParameterGroupParameterArgs{
					Name:  pulumi.String("max_connections"),
					Value: pulumi.String("1000"),
				},
			},
			Tags: pulumi.StringMap{
				"Name":    pulumi.String(fmt.Sprintf("%s-aurora-instance-pg", projectName)),
				"Project": pulumi.String(projectName),
			},
		})
		if err != nil {
			return err
		}

		// Create Aurora Cluster
		cluster, err := rds.NewCluster(ctx, fmt.Sprintf("%s-aurora-cluster", projectName), &rds.ClusterArgs{
			ClusterIdentifier:              pulumi.String(fmt.Sprintf("%s-aurora-cluster", projectName)),
			Engine:                         pulumi.String("aurora-mysql"),
			EngineVersion:                  pulumi.String(engineVersion),
			DatabaseName:                   pulumi.String(dbName),
			MasterUsername:                 pulumi.String(dbUsername),
			MasterPassword:                 dbPassword,
			DbSubnetGroupName:              dbSubnetGroup.Name,
			VpcSecurityGroupIds:            pulumi.StringArray{auroraSecurityGroupId},
			DbClusterParameterGroupName:    clusterParameterGroup.Name,
			BackupRetentionPeriod:          pulumi.Int(7),
			PreferredBackupWindow:          pulumi.String("03:00-04:00"),
			PreferredMaintenanceWindow:     pulumi.String("mon:04:00-mon:05:00"),
			EnabledCloudwatchLogsExports:   pulumi.StringArray{
				pulumi.String("error"),
				pulumi.String("general"),
				pulumi.String("slowquery"),
			},
			StorageEncrypted:               pulumi.Bool(true),
			ApplyImmediately:               pulumi.Bool(true),
			SkipFinalSnapshot:              pulumi.Bool(true),
			Tags: pulumi.StringMap{
				"Name":    pulumi.String(fmt.Sprintf("%s-aurora-cluster", projectName)),
				"Project": pulumi.String(projectName),
			},
		})
		if err != nil {
			return err
		}

		// Create Aurora Writer Instance
		writerInstance, err := rds.NewClusterInstance(ctx, fmt.Sprintf("%s-writer-instance", projectName), &rds.ClusterInstanceArgs{
			Identifier:              pulumi.String(fmt.Sprintf("%s-writer-instance", projectName)),
			ClusterIdentifier:       cluster.ID(),
			InstanceClass:           pulumi.String(instanceClass),
			Engine:                  pulumi.String("aurora-mysql"),
			EngineVersion:           pulumi.String(engineVersion),
			DbParameterGroupName:    instanceParameterGroup.Name,
			PubliclyAccessible:      pulumi.Bool(false),
			AutoMinorVersionUpgrade: pulumi.Bool(false),
			PerformanceInsightsEnabled: pulumi.Bool(true),
			PerformanceInsightsRetentionPeriod: pulumi.Int(7),
			Tags: pulumi.StringMap{
				"Name":    pulumi.String(fmt.Sprintf("%s-writer-instance", projectName)),
				"Project": pulumi.String(projectName),
				"Role":    pulumi.String("writer"),
			},
		})
		if err != nil {
			return err
		}

		// Create Aurora Reader Instance
		readerInstance, err := rds.NewClusterInstance(ctx, fmt.Sprintf("%s-reader-instance", projectName), &rds.ClusterInstanceArgs{
			Identifier:              pulumi.String(fmt.Sprintf("%s-reader-instance", projectName)),
			ClusterIdentifier:       cluster.ID(),
			InstanceClass:           pulumi.String(instanceClass),
			Engine:                  pulumi.String("aurora-mysql"),
			EngineVersion:           pulumi.String(engineVersion),
			DbParameterGroupName:    instanceParameterGroup.Name,
			PubliclyAccessible:      pulumi.Bool(false),
			AutoMinorVersionUpgrade: pulumi.Bool(false),
			PerformanceInsightsEnabled: pulumi.Bool(true),
			PerformanceInsightsRetentionPeriod: pulumi.Int(7),
			Tags: pulumi.StringMap{
				"Name":    pulumi.String(fmt.Sprintf("%s-reader-instance", projectName)),
				"Project": pulumi.String(projectName),
				"Role":    pulumi.String("reader"),
			},
		}, pulumi.DependsOn([]pulumi.Resource{writerInstance}))
		if err != nil {
			return err
		}

		// Export outputs
		ctx.Export("clusterIdentifier", cluster.ClusterIdentifier)
		ctx.Export("clusterArn", cluster.Arn)
		ctx.Export("clusterEndpoint", cluster.Endpoint)
		ctx.Export("clusterReaderEndpoint", cluster.ReaderEndpoint)
		ctx.Export("clusterPort", cluster.Port)
		ctx.Export("databaseName", cluster.DatabaseName)
		ctx.Export("masterUsername", cluster.MasterUsername)
		ctx.Export("engineVersion", cluster.EngineVersion)
		ctx.Export("writerInstanceId", writerInstance.ID())
		ctx.Export("readerInstanceId", readerInstance.ID())
		ctx.Export("writerInstanceEndpoint", writerInstance.Endpoint)
		ctx.Export("readerInstanceEndpoint", readerInstance.Endpoint)

		return nil
	})
}
