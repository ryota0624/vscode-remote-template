package main

import (
	"encoding/base64"
	"fmt"
	"os"

	"github.com/Fiddler25/cdk-go/utils"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	ec2 "github.com/aws/aws-cdk-go/awscdk/v2/awsec2"
	iam "github.com/aws/aws-cdk-go/awscdk/v2/awsiam"

	jsii "github.com/aws/jsii-runtime-go"

	"github.com/aws/constructs-go/constructs/v10"
)

type CdkStackProps struct {
	awscdk.StackProps
}

func NewCdkStack(scope constructs.Construct, id string, props *CdkStackProps) awscdk.Stack {
	var sprops awscdk.StackProps
	if props != nil {
		sprops = props.StackProps
	}
	stack := awscdk.NewStack(scope, &id, &sprops)

	CdkEc2(scope, &CdkEc2Props{
		Stack:      stack,
		StackProps: props.StackProps,
		Vpc: ec2.Vpc_FromLookup(scope, jsii.String("vscode-server-vpc"), &ec2.VpcLookupOptions{
			VpcId: jsii.String("TODO:"),
		}),
		Subnet: ec2.Subnet_FromSubnetId(scope, jsii.String("vscode-subnet"), jsii.String("TODO:")),
	})

	return stack
}

type CdkEc2Props struct {
	awscdk.StackProps
	awscdk.Stack
	Vpc    ec2.IVpc
	Subnet ec2.ISubnet
}

func CdkIAM(scope constructs.Construct, props awscdk.StackProps) iam.Role {
	role := iam.NewRole(scope, jsii.String("iamrole-vscode-server"), &iam.RoleProps{
		RoleName: jsii.String("iamrole-vscode-server"),
	})
	role.AddManagedPolicy(iam.ManagedPolicy_FromAwsManagedPolicyName(jsii.String("AmazonSSMManagedInstanceCore")))
	policy := iam.NewPolicy(scope, jsii.String("iamrole-vscode-server-allow-ssm-ssh"), &iam.PolicyProps{
		PolicyName: jsii.String("iamrole-vscode-server-allow-ssm-ssh"),
	})

	policy.AddStatements(iam.NewPolicyStatement(&iam.PolicyStatementProps{
		Effect:    iam.Effect_ALLOW,
		Actions:   jsii.Strings("ssm:StartSession"),
		Resources: jsii.Strings(fmt.Sprintf("arn:aws:ec2:%s:%s:instance/*", *props.Env.Region, *props.Env.Account), "arn:aws:ssm:*:*:document/AWS-StartSSHSession"),
	},
	))
	role.AttachInlinePolicy(policy)

	return role
}

func CdkEc2(scope constructs.Construct, props *CdkEc2Props) ec2.CfnInstance {
	stack := props.Stack

	sg := ec2.NewCfnSecurityGroup(stack, jsii.String("security-group-vscode"), &ec2.CfnSecurityGroupProps{
		GroupName:        jsii.String("vscode-server-sg"),
		GroupDescription: jsii.String("for vscode server"),
		VpcId:            props.Vpc.VpcId(),
		SecurityGroupIngress: &[]*ec2.CfnSecurityGroup_IngressProperty{
			{
				IpProtocol: jsii.String("tcp"),
				CidrIp:     jsii.String("0.0.0.0/0"),
				FromPort:   jsii.Number(22),
				ToPort:     jsii.Number(22),
			},
			{
				IpProtocol: jsii.String("tcp"),
				CidrIp:     jsii.String("0.0.0.0/0"),
				FromPort:   jsii.Number(80),
				ToPort:     jsii.Number(80),
			},
		},
	})

	role := CdkIAM(scope, props.StackProps)
	amznLinux := ec2.NewAmazonLinuxImage(&ec2.AmazonLinuxImageProps{
		Generation:     ec2.AmazonLinuxGeneration_AMAZON_LINUX,
		Edition:        ec2.AmazonLinuxEdition_STANDARD,
		Virtualization: ec2.AmazonLinuxVirt_HVM,
		Storage:        ec2.AmazonLinuxStorage_GENERAL_PURPOSE,
	})
	// Instance
	return ec2.NewCfnInstance(stack, jsii.String("ec2-instance-vscode"), &ec2.CfnInstanceProps{
		ImageId:            amznLinux.GetImage(scope).ImageId,
		InstanceType:       jsii.String("t2.micro"),
		SubnetId:           props.Subnet.SubnetId(),
		SecurityGroupIds:   jsii.Strings(*sg.AttrGroupId()),
		IamInstanceProfile: role.RoleArn(),
		KeyName:            jsii.String(utils.EnvNames().KeyName),
		UserData:           jsii.String(getUserData()),
		Tags:               &[]*awscdk.CfnTag{{Key: jsii.String("Name"), Value: jsii.String("VSCodeServer")}},
	})
}

func getUserData() string {
	dir, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	f, err := os.ReadFile(dir + "/user_data.sh")
	if err != nil {
		panic(err)
	}

	return base64.StdEncoding.EncodeToString(f)
}

func main() {
	app := awscdk.NewApp(nil)

	NewCdkStack(app, "CdkStack", &CdkStackProps{
		awscdk.StackProps{
			Env: env(),
		},
	})

	app.Synth(nil)
}

// env determines the AWS environment (account+region) in which our stack is to
// be deployed. For more information see: https://docs.aws.amazon.com/cdk/latest/guide/environments.html
func env() *awscdk.Environment {
	// If unspecified, this stack will be "environment-agnostic".
	// Account/Region-dependent features and context lookups will not work, but a
	// single synthesized template can be deployed anywhere.
	//---------------------------------------------------------------------------
	return nil

	// Uncomment if you know exactly what account and region you want to deploy
	// the stack to. This is the recommendation for production stacks.
	//---------------------------------------------------------------------------
	// return &awscdk.Environment{
	//  Account: jsii.String("123456789012"),
	//  Region:  jsii.String("us-east-1"),
	// }

	// Uncomment to specialize this stack for the AWS Account and Region that are
	// implied by the current CLI configuration. This is recommended for dev
	// stacks.
	//---------------------------------------------------------------------------
	// return &awscdk.Environment{
	//  Account: jsii.String(os.Getenv("CDK_DEFAULT_ACCOUNT")),
	//  Region:  jsii.String(os.Getenv("CDK_DEFAULT_REGION")),
	// }
}
