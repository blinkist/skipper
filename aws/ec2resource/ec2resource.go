package ec2resource

import (
	b64 "encoding/base64"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/blinkist/skipper/aws/ec2client"
)

// Ec2resource 'Class' struct
type Ec2resource struct {
	ec2client   *ec2client.Ec2client
	ec2instance *ec2.Instance
	instanceID  *string
}

// New is the constructor of the Ec2resource
func New(argInstanceId *string, ec2client *ec2client.Ec2client) *Ec2resource {
	return &Ec2resource{
		instanceID: argInstanceId,
		ec2client:  ec2client,
	}
}

// GetAttribute returns an instance attribute
func (c *Ec2resource) GetAttribute(attribute string) *ec2.DescribeInstanceAttributeOutput {
	res, err := c.ec2client.DescribeInstanceAttribute(c.instanceID, &attribute)
	if err != nil {
		fmt.Println(err)
		panic("Could not get attribute")
	}
	return res
}

// RefreshInstance reloads the instance
func (c *Ec2resource) RefreshInstance() {
	var err error
	c.ec2instance, err = c.ec2client.DescribeInstance(c.instanceID)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// GetUserData returns Base64-decoded User-Data
func (c *Ec2resource) GetUserData() *string {
	res := c.GetAttribute("userData")

	userdatabytes, err := b64.StdEncoding.DecodeString(*res.UserData.Value)
	userdata := string(userdatabytes)
	if err != nil {
		panic("Could not get Userdata")
	}

	return &userdata
}

// GetIamInstanceProfile returns the Instance Profile ARN
func (c *Ec2resource) GetIamInstanceProfile() *string {
	return c.ec2instance.IamInstanceProfile.Arn
}

// GetImageID returns the ImageID
func (c *Ec2resource) GetImageID() *string {
	return c.ec2instance.ImageId
}

// GetBlockDeviceMapping returns the isntance's Block Device Mapping
func (c *Ec2resource) GetBlockDeviceMapping() []*ec2.InstanceBlockDeviceMapping {
	return c.ec2instance.BlockDeviceMappings
}

// GetSecurityGroupIDS returns the VPC Security group ids
func (c *Ec2resource) GetSecurityGroupIDS() []*string {
	var ids []*string
	for _, v := range c.ec2instance.SecurityGroups {
		ids = append(ids, v.GroupId)
	}
	return ids
}

// UpdateServiceWithTaskDefinition returns the SubnetID of the instance
func (c *Ec2resource) GetSubnetID() *string {
	return c.ec2instance.SubnetId
}
