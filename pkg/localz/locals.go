package localz

import (
	"fmt"
	"github.com/plantoncloud/planton-cloud-apis/zzgo/cloud/planton/apis/code2cloud/v1/aws/awsvpc"
	"github.com/plantoncloud/planton-cloud-apis/zzgo/cloud/planton/apis/commons/apiresource/enums/apiresourcekind"
	"github.com/plantoncloud/pulumi-module-golang-commons/pkg/provider/aws/awstagkeys"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"sort"
	"strconv"
)

type SubnetName string
type SubnetCidr string
type AvailabilityZone string

type Locals struct {
	AwsVpc             *awsvpc.AwsVpc
	AwsTags            map[string]string
	PrivateAzSubnetMap map[AvailabilityZone]map[SubnetName]SubnetCidr
	PublicAzSubnetMap  map[AvailabilityZone]map[SubnetName]SubnetCidr
}

func Initialize(ctx *pulumi.Context, stackInput *awsvpc.AwsVpcStackInput) *Locals {
	locals := &Locals{}

	//assign value for the locals variable to make it available across the project
	locals.AwsVpc = stackInput.Target

	locals.AwsTags = map[string]string{
		awstagkeys.Resource:     strconv.FormatBool(true),
		awstagkeys.Organization: locals.AwsVpc.Spec.EnvironmentInfo.OrgId,
		awstagkeys.Environment:  locals.AwsVpc.Spec.EnvironmentInfo.EnvId,
		awstagkeys.ResourceKind: apiresourcekind.ApiResourceKind_aws_vpc.String(),
		awstagkeys.ResourceId:   locals.AwsVpc.Metadata.Id,
	}

	locals.PrivateAzSubnetMap = GetPrivateAzSubnetMap(locals.AwsVpc)
	locals.PublicAzSubnetMap = GetPublicAzSubnetMap(locals.AwsVpc)

	return locals
}

func GetPrivateAzSubnetMap(awsVpc *awsvpc.AwsVpc) map[AvailabilityZone]map[SubnetName]SubnetCidr {
	privateAzSubnetMap := make(map[AvailabilityZone]map[SubnetName]SubnetCidr, 0)

	for azIndex, az := range awsVpc.Spec.AvailabilityZones {
		for subnetIndex := 0; subnetIndex < int(awsVpc.Spec.SubnetsPerAvailabilityZone); subnetIndex++ {
			//build private subnet name
			privateSubnetName := fmt.Sprintf("private-subnet-%s-%d", az, subnetIndex)
			//calculate private subnet cidr
			privateSubnetCidr := fmt.Sprintf("10.0.%d.0/%d", 100+azIndex*10+subnetIndex, awsVpc.Spec.SubnetSize)

			// Initialize the map for this AvailabilityZone if it doesn't exist
			if privateAzSubnetMap[AvailabilityZone(az)] == nil {
				privateAzSubnetMap[AvailabilityZone(az)] = make(map[SubnetName]SubnetCidr)
			}

			//add private subnet to the locals map
			privateAzSubnetMap[AvailabilityZone(az)][SubnetName(privateSubnetName)] = SubnetCidr(privateSubnetCidr)
		}
	}
	return privateAzSubnetMap
}

func GetPublicAzSubnetMap(awsVpc *awsvpc.AwsVpc) map[AvailabilityZone]map[SubnetName]SubnetCidr {
	publicAzSubnetMap := make(map[AvailabilityZone]map[SubnetName]SubnetCidr, 0)

	for azIndex, az := range awsVpc.Spec.AvailabilityZones {
		for subnetIndex := 0; subnetIndex < int(awsVpc.Spec.SubnetsPerAvailabilityZone); subnetIndex++ {
			//build public subnet name
			publicSubnetName := fmt.Sprintf("public-subnet-%s-%d", az, subnetIndex)
			//calculate public subnet cidr
			publicSubnetCidr := fmt.Sprintf("10.0.%d.0/%d", azIndex*10+subnetIndex, awsVpc.Spec.SubnetSize)
			// Initialize the map for this AvailabilityZone if it doesn't exist
			if publicAzSubnetMap[AvailabilityZone(az)] == nil {
				publicAzSubnetMap[AvailabilityZone(az)] = make(map[SubnetName]SubnetCidr)
			}
			//add public subnet to the locals map
			publicAzSubnetMap[AvailabilityZone(az)][SubnetName(publicSubnetName)] = SubnetCidr(publicSubnetCidr)
		}
	}
	return publicAzSubnetMap
}

func GetSortedAzKeys(azSubnetMap map[AvailabilityZone]map[SubnetName]SubnetCidr) []string {
	keys := make([]string, 0, len(azSubnetMap))
	for k := range azSubnetMap {
		keys = append(keys, string(k))
	}

	sort.Strings(keys)

	return keys
}

func GetSortedSubnetNameKeys(subnetMap map[SubnetName]SubnetCidr) []string {
	keys := make([]string, 0, len(subnetMap))
	for k := range subnetMap {
		keys = append(keys, string(k))
	}

	sort.Strings(keys)

	return keys
}
