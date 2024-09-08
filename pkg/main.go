package pkg

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/plantoncloud/aws-vpc-pulumi-module/pkg/outputs"
	"github.com/plantoncloud/planton-cloud-apis/zzgo/cloud/planton/apis/code2cloud/v1/aws/awsvpc"
	"github.com/plantoncloud/pulumi-module-golang-commons/pkg/datatypes/stringmaps"
	"github.com/plantoncloud/pulumi-module-golang-commons/pkg/datatypes/stringmaps/convertstringmaps"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/ec2"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type ResourceStack struct {
	StackInput *awsvpc.AwsVpcStackInput
}

func (s *ResourceStack) Resources(ctx *pulumi.Context) error {
	locals := initializeLocals(ctx, s.StackInput)

	awsCredential := s.StackInput.AwsCredential

	//create aws provider using the credentials from the input
	awsProvider, err := aws.NewProvider(ctx,
		"classic-provider",
		&aws.ProviderArgs{
			AccessKey: pulumi.String(awsCredential.Spec.AccessKeyId),
			SecretKey: pulumi.String(awsCredential.Spec.SecretAccessKey),
			Region:    pulumi.String(awsCredential.Spec.Region),
		})
	if err != nil {
		return errors.Wrap(err, "failed to create aws provider")
	}

	// create vpc
	createdVpc, err := ec2.NewVpc(ctx,
		locals.AwsVpc.Metadata.Name,
		&ec2.VpcArgs{
			CidrBlock:          pulumi.String(locals.AwsVpc.Spec.VpcCidr),
			EnableDnsSupport:   pulumi.Bool(locals.AwsVpc.Spec.IsDnsSupportEnabled),
			EnableDnsHostnames: pulumi.Bool(locals.AwsVpc.Spec.IsDnsHostnamesEnabled),
			Tags: convertstringmaps.ConvertGoStringMapToPulumiStringMap(
				stringmaps.AddEntry(locals.AwsTags, "Name", locals.AwsVpc.Metadata.Name)),
		}, pulumi.Provider(awsProvider))
	if err != nil {
		return errors.Wrap(err, "failed to create vpc")
	}

	//add vpc id to outputs
	ctx.Export(outputs.VpcId, createdVpc.ID())

	// internet gateway for public subnets
	createdInternetGateway, err := ec2.NewInternetGateway(ctx,
		locals.AwsVpc.Metadata.Name,
		&ec2.InternetGatewayArgs{
			VpcId: createdVpc.ID(),
			Tags: convertstringmaps.ConvertGoStringMapToPulumiStringMap(
				stringmaps.AddEntry(locals.AwsTags, "Name", locals.AwsVpc.Metadata.Name)),
		}, pulumi.Parent(createdVpc))
	if err != nil {
		return errors.Wrap(err, "failed to create internet-gateway")
	}

	//add internet-gateway id to outputs
	ctx.Export(outputs.InternetGatewayId, createdInternetGateway.ID())

	// public route table for internet access
	createdPublicRouteTable, err := ec2.NewRouteTable(ctx,
		"public-route-table",
		&ec2.RouteTableArgs{
			VpcId: createdVpc.ID(),
			Tags: convertstringmaps.ConvertGoStringMapToPulumiStringMap(
				stringmaps.AddEntry(locals.AwsTags, "Name",
					fmt.Sprintf("%s-public", locals.AwsVpc.Metadata.Name))),
			Routes: ec2.RouteTableRouteArray{
				&ec2.RouteTableRouteArgs{
					CidrBlock: pulumi.String(vars.AllowAllCidrBlock),
					GatewayId: createdInternetGateway.ID(),
				},
			},
		}, pulumi.Parent(createdInternetGateway))
	if err != nil {
		return errors.Wrap(err, "failed to created route-table for public internet access")
	}

	createdPrivateSubnets, createdPublicSubnets, err := subnets(ctx, locals, createdVpc)
	if err != nil {
		return errors.Wrap(err, "failed to create subnets")
	}

	// associate route table with public subnets
	for i, createdPublicSubnet := range createdPublicSubnets {
		_, err := ec2.NewRouteTableAssociation(ctx,
			fmt.Sprintf("public-route-assoc-%d", i),
			&ec2.RouteTableAssociationArgs{
				RouteTableId: createdPublicRouteTable.ID(),
				SubnetId:     createdPublicSubnet.ID(),
			})
		if err != nil {
			return errors.Wrap(err, "error associating route table with public subnet")
		}
	}

	// create nat gateways for each private subnet
	if locals.AwsVpc.Spec.IsNatGatewayEnabled {
		if err := natGateways(ctx, awsProvider, createdVpc, createdPrivateSubnets); err != nil {
			return errors.Wrap(err, "failed to create nat gateways")
		}
	}

	return nil
}
