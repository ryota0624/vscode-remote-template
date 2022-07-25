package main

import (
	"encoding/base64"
	"fmt"
	"os"

	"github.com/Fiddler25/cdk-go/utils"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	ec2 "github.com/aws/aws-cdk-go/awscdk/v2/awsec2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsevents"

	iam "github.com/aws/aws-cdk-go/awscdk/v2/awsiam"

	jsii "github.com/aws/jsii-runtime-go"

	"github.com/aws/constructs-go/constructs/v10"
)

type CdkStackProps struct {
	Environment
	awscdk.StackProps
}

func NewCdkStack(scope constructs.Construct, id string, props *CdkStackProps) awscdk.Stack {
	var sprops awscdk.StackProps
	if props != nil {
		sprops = props.StackProps
	}
	stack := awscdk.NewStack(scope, &id, &sprops)
	ec2 := CdkVSCodeServerEc2(stack, &CdkEc2Props{
		StackProps:   props.StackProps,
		VpcId:        props.Environment.VpcId,
		SubnetId:     props.Environment.SubnetId,
		InstanceSize: props.Environment.InstanceSize,
	})

	EC2StopEventBridge(stack, &EC2StopEventBridgeProps{
		StopTargetEC2Arn: *ec2.Ref(),
	})

	return stack
}

type CdkEc2Props struct {
	awscdk.StackProps
	VpcId        *string
	SubnetId     *string
	InstanceSize *string
}

type EC2StopEventBridgeProps struct {
	StopTargetEC2Arn string
}

func EC2StopEventBridge(scope constructs.Construct, props *EC2StopEventBridgeProps) {
	role := iam.NewRole(scope, jsii.String("iamrole-stop-vscode-server"), &iam.RoleProps{
		RoleName:  jsii.String("iamrole-stop-vscode-server"),
		AssumedBy: iam.NewServicePrincipal(jsii.String("events.amazon.com"), &iam.ServicePrincipalOpts{}),
	})
	role.AddManagedPolicy(iam.ManagedPolicy_FromAwsManagedPolicyName(jsii.String("AmazonSSMAutomationRole")))

	awsevents.NewCfnRule(scope, jsii.String("vscode-server-stop"), &awsevents.CfnRuleProps{
		ScheduleExpression: jsii.String("0 5 * * ? *"),
		Name:               jsii.String("vscode-server-stop"),
		Description:        jsii.String("to daily stop vscode server ec2 instance"),
		Targets: []interface{}{
			map[string]string{
				"Arn":     "arn:aws:ssm:ap-northeast-1::automation-definition/AWS-StopEC2Instance:$DEFAULT",
				"RoleArn": *role.RoleArn(),
				"Input":   props.StopTargetEC2Arn,
				"Id":      "TargetStopVsCodeEC2Instance",
			},
		},
	})
}

func CdkVSCodeServerIAM(scope constructs.Construct, props awscdk.StackProps) iam.Role {
	role := iam.NewRole(scope, jsii.String("iamrole-vscode-server"), &iam.RoleProps{
		RoleName:  jsii.String("iamrole-vscode-server"),
		AssumedBy: iam.NewServicePrincipal(jsii.String("ec2.amazonaws.com"), &iam.ServicePrincipalOpts{}),
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

func CdkVSCodeServerEc2(scope constructs.Construct, props *CdkEc2Props) ec2.CfnInstance {

	sg := ec2.NewCfnSecurityGroup(scope, jsii.String("security-group-vscode"), &ec2.CfnSecurityGroupProps{
		GroupName:            jsii.String("vscode-server-sg"),
		GroupDescription:     jsii.String("for vscode server"),
		VpcId:                props.VpcId,
		SecurityGroupIngress: &[]*ec2.CfnSecurityGroup_IngressProperty{},
	})

	role := CdkVSCodeServerIAM(scope, props.StackProps)
	amznLinux := ec2.NewAmazonLinuxImage(&ec2.AmazonLinuxImageProps{
		Generation:     ec2.AmazonLinuxGeneration_AMAZON_LINUX,
		Edition:        ec2.AmazonLinuxEdition_STANDARD,
		Virtualization: ec2.AmazonLinuxVirt_HVM,
		Storage:        ec2.AmazonLinuxStorage_GENERAL_PURPOSE,
	})
	// Instance
	return ec2.NewCfnInstance(scope, jsii.String("ec2-instance-vscode"), &ec2.CfnInstanceProps{
		ImageId:            amznLinux.GetImage(scope).ImageId,
		InstanceType:       props.InstanceSize,
		SubnetId:           props.SubnetId,
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
	environment := env()

	NewCdkStack(app, "CdkStack", &CdkStackProps{
		*environment,
		awscdk.StackProps{
			Env: &environment.Environment,
		},
	})

	app.Synth(nil)
}

// TODO: TODOとついているリソースIDやサイズ指定をenvからできるようにしたい。
// 	EC2インスタンスの自動停止
//  EC2インスタンスのコマンド一発起動
//  budgetアラート

type Environment struct {
	awscdk.Environment
	VpcId        *string
	SubnetId     *string
	InstanceSize *string
}

// env determines the AWS environment (account+region) in which our stack is to
// be deployed. For more information see: https://docs.aws.amazon.com/cdk/latest/guide/environments.html
func env() *Environment {
	// If unspecified, this stack will be "environment-agnostic".
	// Account/Region-dependent features and context lookups will not work, but a
	// single synthesized template can be deployed anywhere.
	//---------------------------------------------------------------------------
	// return nil

	// Uncomment if you know exactly what account and region you want to deploy
	// the stack to. This is the recommendation for production stacks.
	//---------------------------------------------------------------------------
	return &Environment{
		Environment: awscdk.Environment{
			Account: jsii.String(os.Getenv("CDK_DEFAULT_ACCOUNT")),
			Region:  jsii.String(os.Getenv("CDK_DEFAULT_REGION")),
		},
		SubnetId:     jsii.String(os.Getenv("VSCODE_SUBNET_ID")),
		VpcId:        jsii.String(os.Getenv("VSCODE_VPC_ID")),
		InstanceSize: jsii.String(os.Getenv("VSCODE_INSTANCE_SIZE")),
	}
}
