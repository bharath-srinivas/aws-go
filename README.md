# AWS Go
[![Build Status](https://travis-ci.org/bharath-srinivas/aws-go.svg?branch=master)](https://travis-ci.org/bharath-srinivas/aws-go)
[![GoDoc](https://godoc.org/github.com/bharath-srinivas/aws-go?status.svg)](https://godoc.org/github.com/bharath-srinivas/aws-go)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

AWS Go is a CLI tool for managing [AWS](https://aws.amazon.com) services without the need
to login to the AWS console, built to be fast and easy to use. Currently AWS Go supports services like
EC2, Lambda and RDS only. Support for more services will be added later.

## Installation

Currently AWS Go is available only for Linux amd64 architecture. Support for other operating systems and 
architectures will be added later once the tool is stable.

On Linux run the following command to install aws-go:

```
curl -sL https://raw.githubusercontent.com/bharath-srinivas/aws-go/master/setup_aws_go | sudo -E bash -
```

If already installed, upgrade with:

```bash
$ sudo aws-go upgrade
```

## AWS credentials

AWS Go requires AWS Credentials to perform operations and to manage resources. You can provide your credentials
to AWS Go and manage them by using the `env` command.

### Managing environments

You can manage your environment profiles and create new profiles with the `env` command. It stores your config
so that you can transition between different profiles seamlessly without the need to enter your credentials
every time you switch to different environment.

For creating a new profile, use the following command:

```bash
$ aws-go env create --profile production --region us-west-1
```

You'll require the following details for creating a new profile:

* `AWS Access Key ID` your AWS account's access key
* `AWS Secret Access Key` your AWS account's secret key

You can switch between environments with the following command:

```bash
$ aws-go env use --profile staging --region eu-west-1
```

In both the above commands, the `--region` flag is `optional` and the default value will be `us-east-1` if the value
for the flag is not provided. 

### Listing profiles

For listing all the stored profiles:

```bash
$ aws-go env --list
```

### Deleting profile

For deleting a profile:

```bash
$ aws-go env --delete staging
``` 

### Minimum IAM policy

Below is the [AWS IAM](https://aws.amazon.com/iam) policy which provides the minimum required permissions for `aws-go`
to function.

For EC2:

```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": "ec2:Describe*",
            "Resource": "*"
        }
    ]
}
```

For Lambda:

```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "iam:GetRole",
                "iam:GetRolePolicy",
                "iam:ListAttachedRolePolicies",
                "iam:ListRolePolicies",
                "iam:ListRoles",
                "lambda:Get*",
                "lambda:List*"
            ],
            "Resource": "*"
        }
    ]
}
```

For RDS:

```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Action": [
                "rds:Describe*",
                "rds:ListTagsForResource",
                "ec2:DescribeAccountAttributes",
                "ec2:DescribeAvailabilityZones",
                "ec2:DescribeInternetGateways",
                "ec2:DescribeSecurityGroups",
                "ec2:DescribeSubnets",
                "ec2:DescribeVpcAttribute",
                "ec2:DescribeVpcs"
            ],
            "Effect": "Allow",
            "Resource": "*"
        }
    ]
}
```

#### IAM policy for starting and stopping instances

The following additional IAM policy is needed to start and stop the EC2 instances using aws-go.

```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Action": "ec2:*",
            "Effect": "Allow",
            "Resource": "*"
        }
    ]
}
```

#### IAM policy for invoking Lambda functions

The following additional IAM policy is required to invoke Lambda functions using aws-go.

**Note:** The following IAM policy provides permissions to a minimalistic amount of AWS resources and may vary 
according to the type of Lambda function your're invoking as your function might require access to additional
resources like EC2, CloudWatch, S3 etc. Please refer to the [official documentation](https://docs.aws.amazon.com/IAM/latest/UserGuide/introduction.html)
for more information on how to set the required policies. 

```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "cloudwatch:*",
                "iam:CreateRole",
                "iam:CreatePolicy",
                "iam:AttachRolePolicy",
                "iam:PassRole",
                "lambda:*",
                "logs:*",
                "s3:*"
            ],
            "Resource": "*"
        }
    ]
}
``` 

## Getting started

**Note:** Before using aws-go, [AWS credentials](#aws-credentials) are required for using the CLI.

### Listing EC2 instances

For listing all the EC2 instances in the current selected profile, you just have to run `list`. This will list all the
available EC2 instances in a table like structure excluding the ones that are being terminated or already terminated.
You can get entire info about all the instances in `JSON` format using the `-a` flag. You can also apply filters to the
list with the `--filters` or `-f` flag. The filtering functionality is supported only on normal table listing and not 
on the `-a` flag.

#### Supported filters:
The following filters are supported by the list command. It's important to note that with the `--filters` flag, you
cannot search for multiple instance names or multiple availability zones etc. In that case you can use the `JSON` file
which allows you to filter based on multiple values. 

Note that every filter is case insensitive:

* `name` instance name
* `id` instance ID
* `state` instance state
* `type` instance type
* `az` availability zone of the instance

#### Example filters file:
     
```json
 [
   {
     "name": "name",
     "values": ["web"]
   },
   {
     "name": "az",
     "values": ["us-east-1a", "us-east-1d"]
   }
 ]
```

#### Example

List all the available EC2 instances in a table format:

```bash
$ aws-go ec2 list
```

Apply filters to the list:

```bash
$ aws-go ec2 list --filters name=web,az=us-east-1a
```

Applying filters from a `JSON` file:

```bash
$ aws-go ec2 list -F filters.json
```

Listing everything in `JSON` format:

```bash
$ aws-go ec2 list --all
```

Piping `JSON` to a `JSON` file:

```bash
$ aws-go ec2 list --all > ec2.json
```

Performing `less` on `JSON` output:

```bash
$ aws-go ec2 list --all | less
```

### Starting EC2 instances

AWS Go currently supports starting or stopping a single instance at a time. For starting an instance, you have to use
the `start` command along with the `instance-id` of the instance you want to start as the argument.

#### Example

Starting an EC2 instance:

```bash
$ aws-go ec2 start i-0a12b345c678de
```

Performing a `--dry-run` operation:

```bash
$ aws-go ec2 start --dry-run i-0a12b345c678de
```

### Stopping EC2 instances

To stop an EC2 instance, use the `stop` command along with the `instance-id` of the instance you want to start 
as the argument.

#### Example

Stopping an EC2 instance:

```bash
$ aws-go ec2 stop i-0a12b345c678de
```

Performing a `--dry-run` operation:

```bash
$ aws-go ec2 stop --dry-run i-0a12b345c678de
```

### Listing Lambda functions

AWS Go lists all the available Lambda functions and their configurations in a human friendly terminal output.

#### Example

Listing the Lambda functions and their configurations:

```bash
$ aws-go lambda list
```

### Invoking Lambda functions

AWS Go allows you to invoke the specified AWS Lambda function from the command-line and it returns the status code of 
the function call. It's important to note that `invoke` command invokes the `$LATEST` version of the lambda function
available with RequestResponse invocation type.

#### Example

Invoking a Lambda function:

```bash
$ aws-go lambda invoke testLambdaFunction
```

### Listing RDS instances

AWS Go lists only the available RDS instances excluding the ones that are being terminated or already terminated. 
AWS Go provides only the basic information about RDS instances since the terminal cannot accommodate all the information
about RDS instances. This might be improved in the future.

#### Example

Listing the RDS instances in a table:

```bash
$ aws-go rds list
```