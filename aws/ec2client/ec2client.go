package ec2client

import (
	b64 "encoding/base64"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

var (
	instance *Ec2client
	once     sync.Once
)

// Ec2client type
type Ec2client struct {
	session     *session.Session
	svc         *ec2.EC2
	creds       *credentials.Credentials
	logger      *log.Logger
	clusterArns map[string]string
	timeout     int
}

// New Constructor, takes the default region
func New() *Ec2client {
	sess := session.New()
	svc := ec2.New(sess)
	logger := log.New(os.Stderr, " - ", log.LstdFlags)

	return &Ec2client{
		clusterArns: nil,
		svc:         svc,
		logger:      logger,
		session:     sess,
		timeout:     300,
	}
}

// GetInstance Singleton Method to retrieve the
func GetInstance() *Ec2client {
	once.Do(func() {
		instance = New()
	})
	return instance
}

// CreateKeypair creates a Keypair for *string name
func (c *Ec2client) CreateKeypair(pairName *string) (*ec2.CreateKeyPairOutput, error) {
	result, err := c.svc.CreateKeyPair(&ec2.CreateKeyPairInput{
		KeyName: aws.String(*pairName),
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok && aerr.Code() == "InvalidKeyPair.Duplicate" {
			return nil, fmt.Errorf("Keypair %q already exists", err.Error())
		}
		return nil, fmt.Errorf("Unable to create key pair: %s, %v", *pairName, err)
	}
	return result, nil
}

// KeypairExists Checks if a keypair exists
func (c *Ec2client) KeypairExists(pairName *string) bool {
	req := &ec2.DescribeKeyPairsInput{
		KeyNames: []*string{aws.String(*pairName)},
	}

	resp, err := c.svc.DescribeKeyPairs(req)
	if err != nil {
		return false
	}

	for _, keyPair := range resp.KeyPairs {
		if *keyPair.KeyName == *pairName {
			return true
		}
	}
	return false
}

// DeleteKeypair Deletes a keypair
func (c *Ec2client) DeleteKeypair(pairName *string) {
	_, err := c.svc.DeleteKeyPair(&ec2.DeleteKeyPairInput{
		KeyName: aws.String(*pairName),
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok && aerr.Code() == "InvalidKeyPair.Duplicate" {
			fmt.Printf("Key pair %q does not exist.", pairName)
			os.Exit(1)
		}
		fmt.Printf("Unable to delete key pair: %s, %v", *pairName, err)
		os.Exit(1)
	}
	fmt.Printf("Succesfully deleted %s\n", *pairName)
}

// DescribeInstanceAttribute Returns inputed attribute for inputed instance
func (c *Ec2client) DescribeInstanceAttribute(instance *string, attribute *string) (*ec2.DescribeInstanceAttributeOutput, error) {

	input := &ec2.DescribeInstanceAttributeInput{
		Attribute:  aws.String(*attribute),
		InstanceId: aws.String(*instance),
	}

	result, err := c.svc.DescribeInstanceAttribute(input)
	if err != nil {
		return nil, fmt.Errorf("Error describing instance: %v", err)
	}
	return result, nil
}

// GetInstancesWithTagName Returns instances by TagName
func (c *Ec2client) GetInstancesWithTagName(name *string) []*ec2.Instance {

	params := &ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			&ec2.Filter{
				Name: aws.String("tag:Name"),
				Values: []*string{
					aws.String(*name),
				},
			},
		},
	}

	res, _ := c.svc.DescribeInstances(params)
	if len(res.Reservations) == 0 {
		return []*ec2.Instance{}
	}

	return res.Reservations[0].Instances
}

// DescribeInstances returns a list of isntances for a list of intanceIds
func (c *Ec2client) DescribeInstances(instances []*string) []*ec2.Instance {

	params := &ec2.DescribeInstancesInput{
		InstanceIds: instances,
	}

	res, err := c.svc.DescribeInstances(params)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	if res.NextToken != nil {
		fmt.Println("Pagination not implemented")
		os.Exit(1)
	}
	allInstances := make([]*ec2.Instance, 0)

	for _, r := range res.Reservations {
		for _, i := range r.Instances {
			allInstances = append(allInstances, i)
		}
	}
	return allInstances
}

// StartInstanceInput holds the input for StartInstance
type StartInstanceInput struct {
	_ struct{} `type:"structure"`

	IamInstanceProfileArn *string
	// The ID of the AMI, which you can get by calling DescribeImages.
	//
	// ImageId is a required field
	ImageID *string `type:"string" required:"true"`

	// The instance type. For more information, see Instance Types (http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/instance-types.html)
	// in the Amazon Elastic Compute Cloud User Guide.
	//
	// Default: m1.small
	InstanceType *string `type:"string" enum:"InstanceType"`

	// The name of the key pair. You can create a key pair using CreateKeyPair or
	// ImportKeyPair.
	//
	// If you do not specify a key pair, you can't connect to the instance unless
	// you choose an AMI that is configured to allow users another way to log in.
	KeyName *string `type:"string"`

	// One or more security group IDs. You can create a security group using CreateSecurityGroup.
	//
	// Default: Amazon EC2 uses the default security group.
	SecurityGroupIds []*string `locationName:"SecurityGroupId" locationNameList:"SecurityGroupId" type:"list"`

	SubnetID *string `type:"string"`

	TagValue *string `type:"string"`

	// The user data to make available to the instance. For more information, see
	// Running Commands on Your Linux Instance at Launch (http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/user-data.html)
	// (Linux) and Adding User Data (http://docs.aws.amazon.com/AWSEC2/latest/WindowsGuide/ec2-instance-metadata.html#instancedata-add-user-data)
	// (Windows). If you are using a command line tool, base64-encoding is performed
	// for you, and you can load the text from a file. Otherwise, you must provide
	// base64-encoded text.
	UserData *string `type:"string"`
}

// StartInstance starts an Instance
func (c *Ec2client) StartInstance(startInstanceInput *StartInstanceInput) (*ec2.Instance, error) {

	encodedUserdata := b64.StdEncoding.EncodeToString([]byte(*startInstanceInput.UserData))

	runResult, err := c.svc.RunInstances(&ec2.RunInstancesInput{
		// An Amazon Linux AMI ID for t2.micro instances in the us-west-2 region

		IamInstanceProfile: &ec2.IamInstanceProfileSpecification{
			Arn: aws.String(*startInstanceInput.IamInstanceProfileArn),
		},
		SecurityGroupIds: startInstanceInput.SecurityGroupIds,
		ImageId:          aws.String(*startInstanceInput.ImageID),
		InstanceType:     aws.String(*startInstanceInput.InstanceType),
		KeyName:          aws.String(*startInstanceInput.KeyName),
		SubnetId:         startInstanceInput.SubnetID,
		MinCount:         aws.Int64(1),
		MaxCount:         aws.Int64(1),
		UserData:         &encodedUserdata,
	})

	if err != nil {
		return nil, fmt.Errorf("Some error happened creating the instance: %s", err)
	}
	c.logger.Println("Created instance", *runResult.Instances[0].InstanceId)

	// Add tags to the created instance
	_, errtag := c.svc.CreateTags(&ec2.CreateTagsInput{
		Resources: []*string{runResult.Instances[0].InstanceId},
		Tags: []*ec2.Tag{
			{
				Key:   aws.String("Name"),
				Value: aws.String(*startInstanceInput.TagValue),
			},
		},
	})
	if errtag != nil {
		return nil, fmt.Errorf("Could not create tags for instance %s: %v", runResult.Instances[0].InstanceId, errtag)

	}

	c.logger.Println("Successfully tagged instance")

	describeInstanceInput := &ec2.DescribeInstancesInput{
		InstanceIds: []*string{runResult.Instances[0].InstanceId},
	}
	c.logger.Println("Waiting until the instance exists")

	waiterr := c.svc.WaitUntilInstanceRunning(describeInstanceInput)
	if waiterr != nil {
		return nil, waiterr
	}

	return runResult.Instances[0], nil
}

// TerminateInstance terminates an instance
func (c *Ec2client) TerminateInstance(instance *ec2.Instance) error {
	_, err := c.svc.TerminateInstances(&ec2.TerminateInstancesInput{
		InstanceIds: []*string{instance.InstanceId},
	})
	return err
}

// DescribeInstance returns the ec2.Instance type for the given Instance ID
func (c *Ec2client) DescribeInstance(instanceId *string) (*ec2.Instance, error) {
	instances := []*string{instanceId}
	retInstances := c.DescribeInstances(instances)
	if len(retInstances) != 1 {
		log.Fatalln("Problem retrieving one instance")
	}
	return retInstances[0], nil
}
