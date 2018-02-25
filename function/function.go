// Package function contains the core functions of aws-go.
package function

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/aws/aws-sdk-go/service/lambda/lambdaiface"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/aws/aws-sdk-go/service/rds/rdsiface"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/aws/aws-sdk-go/service/s3/s3manager/s3manageriface"

	"github.com/bharath-srinivas/aws-go/store"
	"github.com/bharath-srinivas/aws-go/utils"
)

// Invocation type for invoking the Lambda function.
const InvocationType = "RequestResponse"

// filterMap contains the mappings between the filters provided by the user and the actual EC2 filters.
var filterMap = map[string]string{
	"az":    "availability-zone",
	"id":    "instance-id",
	"name":  "tag:Name",
	"state": "instance-state-name",
	"type":  "instance-type",
}

// EC2 represents the AWS EC2 instance fields.
type EC2 struct {
	Name         string   // Name of the EC2 instance
	IDs          []string // List of EC2 instance IDs
	State        string   // Current State of the EC2 instance
	PrivateIP    string   // Private IP address of the EC2 instance
	PublicIP     string   // Public/Elastic IP address of the EC2 instance
	InstanceType string   // EC2 instance type
}

// EC2Service represents the EC2 interface.
type EC2Service struct {
	EC2
	Filters []*ec2.Filter // List of filters to apply for EC2 list
	Service ec2iface.EC2API
}

// Function represents the Lambda function fields.
type Function struct {
	Name        string // Name of the Lambda function
	Description string // Description provided for the Lambda function, if any
	Runtime     string // Runtime of the Lambda function
	Memory      int64  // Memory allocated for the Lambda function
	Timeout     int64  // Timeout set for the Lambda function
	Handler     string // Lambda function handler
	Role        string // IAM role assigned for the Lambda function
	Version     string // Version of the Lambda function
}

// LambdaService represents the Lambda interface.
type LambdaService struct {
	Function
	Service lambdaiface.LambdaAPI
}

// RDS represents the RDS instance fields.
type RDS struct {
	InstanceID    string // RDS instance ID
	Status        string // Current status of the RDS instance
	Endpoint      string // Endpoint of the RDS instance
	InstanceClass string // RDS instance class
	Engine        string // RDS engine
	EngineVersion string // RDS engine version
	MultiAZ       bool   // Multi-AZ availability of the RDS instance
}

// RDSService represents the RDS interface.
type RDSService struct {
	RDS
	Service rdsiface.RDSAPI
}

// S3 represents the S3 bucket fields.
type S3 struct {
	Name         string // S3 bucket name
	CreationDate string // S3 bucket creation date
}

// S3Options represents the optional arguments for S3.
type S3Options struct {
	ContinuationToken string // Continuation token for fetching previous or next set of records
	MaxCount          int64  // Maximum objects to fetch per request
	Prefix            string // Prefix of the S3 objects
}

// S3Downloader represents the Downloader interface.
type S3Downloader struct {
	FileName   string // The file to write the contents of the s3 object.
	Key        string // The S3 object key.
	Downloader s3manageriface.DownloaderAPI
}

// S3Service represents the S3 interface.
type S3Service struct {
	S3
	S3Downloader
	S3Options
	Service s3iface.S3API
}

// NewSession returns an instance of AWS Session.
func NewSession() *session.Session {
	userCreds := store.GetCredentials()

	creds := credentials.NewStaticCredentialsFromCreds(credentials.Value{
		AccessKeyID:     userCreds.AccessId,
		SecretAccessKey: userCreds.SecretKey})

	sess := session.Must(session.NewSessionWithOptions(session.Options{
		Config: aws.Config{Credentials: creds, Region: aws.String(userCreds.Region)},
	}))

	return sess
}

// GetAllInstances gets all the EC2 instances info and returns them as it is.
func (e *EC2Service) GetAllInstances() (*ec2.DescribeInstancesOutput, error) {
	params := &ec2.DescribeInstancesInput{}
	return e.Service.DescribeInstances(params)
}

// SetFilters sets the user provided filter list to the Filters by replacing the user provided filter keys with
// the actual filter keys and returns an error, if any.
func (e *EC2Service) SetFilters(filters []string) error {
	for _, filter := range filters {
		if !strings.Contains(filter, "=") {
			return errors.New("filter error: invalid filter format")
		}

		var filterList ec2.Filter
		userFilter := strings.Split(filter, "=")

		filterName, ok := filterMap[userFilter[0]]
		if !ok {
			return errors.New("filter error: invalid filter key: '" + userFilter[0] + "'")
		}

		filterValues := []string{
			"*" + userFilter[1] + "*",
			"*" + strings.Title(userFilter[1]) + "*",
			"*" + strings.ToLower(userFilter[1]) + "*",
		}

		filterList.Name = aws.String(filterName)
		filterList.Values = aws.StringSlice(filterValues)

		e.Filters = append(e.Filters, &filterList)
	}

	return nil
}

// LoadFiltersFromFile loads the user provided filters that are present in the json file to the Filters by replacing
// them with actual filter keys.
func (e *EC2Service) LoadFiltersFromFile(filtersFile string) error {
	if !strings.Contains(filtersFile, ".json") {
		return errors.New("filter file error: invalid file format")
	}

	fileContent, err := ioutil.ReadFile(filtersFile)
	if err != nil {
		return err
	}

	decoder := json.NewDecoder(strings.NewReader(string(fileContent)))
	decoder.Decode(&e.Filters)

	for _, filter := range e.Filters {
		filterName, ok := filterMap[*filter.Name]
		if !ok {
			return errors.New("filter error: invalid filter key: '" + *filter.Name + "'")
		}

		*filter.Name = filterName
		for _, value := range filter.Values {
			*value = "*" + *value + "*"
			valueTitle := strings.Title(*value)
			valueLower := strings.ToLower(*value)
			filter.Values = append(filter.Values, &valueTitle, &valueLower)
		}
	}

	return nil
}

// GetInstances returns the list of specific fields of AWS EC2 instances as multidimensional slice suitable
// for rendering on a terminal ASCII table.
func (e *EC2Service) GetInstances() ([][]string, error) {
	var params *ec2.DescribeInstancesInput

	if e.Filters != nil {
		params = &ec2.DescribeInstancesInput{
			Filters: e.Filters,
		}
	}

	resp, err := e.Service.DescribeInstances(params)

	if err != nil {
		return nil, err
	}

	var result [][]string
	for _, i := range resp.Reservations {
		var ec2List []string
		for _, t := range i.Instances {
			if *t.State.Name == "terminated" {
				continue
			}

			if t.Tags != nil {
				ec2List = append(ec2List, getInstanceName(t.Tags))
			}

			var publicIP string
			if t.PublicIpAddress != nil {
				publicIP = *t.PublicIpAddress
			}

			ec2List = append(ec2List, *t.InstanceId, *t.State.Name, *t.PrivateIpAddress, publicIP, *t.InstanceType)
		}
		result = append(result, ec2List)
	}
	return result, nil
}

// StartInstances starts the specified instance and returns the state change information of that instance.
func (e *EC2Service) StartInstances(dryRun bool) (*ec2.StartInstancesOutput, error) {
	var instanceIds []*string
	for _, id := range e.IDs {
		instanceIds = append(instanceIds, aws.String(id))
	}

	params := &ec2.StartInstancesInput{
		DryRun:      aws.Bool(dryRun),
		InstanceIds: instanceIds,
	}
	return e.Service.StartInstances(params)
}

// StopInstances stops the specified instance and returns the state change information of that instance.
func (e *EC2Service) StopInstances(dryRun bool) (*ec2.StopInstancesOutput, error) {
	var instanceIds []*string
	for _, id := range e.IDs {
		instanceIds = append(instanceIds, aws.String(id))
	}

	params := &ec2.StopInstancesInput{
		DryRun:      aws.Bool(dryRun),
		InstanceIds: instanceIds,
	}
	return e.Service.StopInstances(params)
}

// GetFunctions returns the list of all the Lambda functions with their configurations.
func (l *LambdaService) GetFunctions() (*lambda.ListFunctionsOutput, error) {
	params := &lambda.ListFunctionsInput{}
	return l.Service.ListFunctions(params)
}

// InvokeFunction invokes the specified function in RequestResponse invocation type and returns the status code.
func (l *LambdaService) InvokeFunction() (*lambda.InvokeOutput, error) {
	params := &lambda.InvokeInput{
		FunctionName:   aws.String(l.Name),
		InvocationType: aws.String(InvocationType),
	}

	return l.Service.Invoke(params)
}

// GetRDSInstances returns the list of specific fields of AWS RDS instances as multidimensional slice suitable
// for rendering on a terminal ASCII table.
func (r *RDSService) GetRDSInstances() ([][]string, error) {
	params := &rds.DescribeDBInstancesInput{}
	resp, err := r.Service.DescribeDBInstances(params)

	if err != nil {
		return nil, err
	}

	var result [][]string
	for _, instance := range resp.DBInstances {
		if *instance.DBInstanceStatus == "terminated" {
			continue
		}

		var rdsList []string
		dbInstanceID := utils.WordWrap(*instance.DBInstanceIdentifier, '-', 2)
		endpoint := utils.WordWrap(*instance.Endpoint.Address, '.', 2)

		engineInfo := *instance.Engine + "/" + *instance.EngineVersion

		rdsList = append(rdsList, dbInstanceID, *instance.DBInstanceStatus, endpoint, *instance.DBInstanceClass,
			engineInfo, strconv.FormatBool(*instance.MultiAZ))

		result = append(result, rdsList)
	}
	return result, nil
}

// GetBuckets returns the list of all the S3 buckets.
func (s *S3Service) GetBuckets() (*s3.ListBucketsOutput, error) {
	params := &s3.ListBucketsInput{}
	return s.Service.ListBuckets(params)
}

// GetObjects returns the list of all the S3 objects from the specified bucket.
func (s *S3Service) GetObjects() (*s3.ListObjectsV2Output, error) {
	var params *s3.ListObjectsV2Input
	params = &s3.ListObjectsV2Input{
		Bucket:  aws.String(s.Name),
		MaxKeys: aws.Int64(s.MaxCount),
	}

	if s.Prefix != "" {
		params.Prefix = aws.String(s.Prefix)
	}

	if s.ContinuationToken != "" {
		params.ContinuationToken = aws.String(s.ContinuationToken)
	}
	return s.Service.ListObjectsV2(params)
}

// DownloadObject downloads a single S3 object from a bucket to the specified location on the system.
func (s *S3Service) DownloadObject() (int64, error) {
	file, err := os.Create(s.FileName)
	if err != nil {
		return 0, err
	}

	params := &s3.GetObjectInput{
		Bucket: aws.String(s.Name),
		Key:    aws.String(s.Key),
	}
	return s.Downloader.Download(file, params)
}

// getInstanceName is a helper function which will return the instance name from the given tag list.
func getInstanceName(tag []*ec2.Tag) string {
	var instanceName string
	for _, tag := range tag {
		if *tag.Key == "Name" {
			instanceName = *tag.Value
		}
	}
	return instanceName
}
