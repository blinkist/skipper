package ssmclient

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"
)

type Ssmclient struct {
	session *session.Session
	svc     *ssm.SSM
	creds   *credentials.Credentials
	logger  *log.Logger
}

type Ssmkeypair struct {
	Key   *string
	Value *string
}

type Ssmkeypairhistory struct {
	Key              *string
	Value            *string
	Version          *int64
	LastModifiedUser *string
	LastModifiedDate *time.Time
}

func New() *Ssmclient {
	sess := session.New()
	svc := ssm.New(sess)

	return &Ssmclient{
		svc:     svc,
		session: sess,
	}
}

func (c *Ssmclient) GetParameters(application *string) ([]*Ssmkeypair, error) {

	withDecryption := true
	mypath := fmt.Sprintf("/application/%s", *application)

	ssmkeypairs := make([]*Ssmkeypair, 0)

	/// START
	var nextToken *string
	for {
		resp, err := c.svc.GetParametersByPath(&ssm.GetParametersByPathInput{
			Path:           &mypath,
			WithDecryption: &withDecryption,
			NextToken:      nextToken,
			Recursive:      aws.Bool(true),
		})

		if err != nil {
			log.Fatalf("ssm:GetParametersByPath failed. (path: %s)\n %v", mypath, err)
		}

		for i := range resp.Parameters {
			name := *resp.Parameters[i].Name
			value := *resp.Parameters[i].Value
			a := Ssmkeypair{
				Key:   &name,
				Value: &value,
			}
			ssmkeypairs = append(ssmkeypairs, &a)
		}

		nextToken = resp.NextToken

		if nextToken == nil || aws.StringValue(nextToken) == "" {

			break
		}
	}

	return ssmkeypairs, nil
}

func (c *Ssmclient) GetParameterHistory(application *string, name *string) ([]*Ssmkeypairhistory, error) {

	withDecryption := true
	fullname := fmt.Sprintf("/application/%s/%s", *application, *name)
	params := &ssm.GetParameterHistoryInput{
		Name:           &fullname,
		WithDecryption: &withDecryption,
	}
	resp, err := c.svc.GetParameterHistory(params)
	if err != nil {
		log.Fatal(err)
	}
	ssmkeypairs := make([]*Ssmkeypairhistory, len(resp.Parameters))

	for i := range resp.Parameters {
		name := *resp.Parameters[i].Name
		value := *resp.Parameters[i].Value
		a := Ssmkeypairhistory{
			Key:              &name,
			Value:            &value,
			Version:          resp.Parameters[i].Version,
			LastModifiedUser: resp.Parameters[i].LastModifiedUser,
			LastModifiedDate: resp.Parameters[i].LastModifiedDate,
		}
		ssmkeypairs[i] = &a
	}

	return ssmkeypairs, nil
}

func (c *Ssmclient) PutParameter(application *string, name *string, value *string) error {
	overwrite := true
	kmskeyid := fmt.Sprintf("alias/application/%s", *application)
	ssmparamname := fmt.Sprintf("/application/%s/%s", *application, *name)
	parameterType := "SecureString"
	input := &ssm.PutParameterInput{
		Name:      &ssmparamname,
		Value:     value,
		KeyId:     &kmskeyid,
		Type:      &parameterType,
		Overwrite: &overwrite,
	}

	_, err := c.svc.PutParameter(input)
	if err != nil {
		panic(fmt.Sprintf("Error authorizing: %s\n", err.Error()))
	}
	return nil
}

func (c *Ssmclient) DeleteParameter(application *string, name *string) error {
	ssmparamname := fmt.Sprintf("/application/%s/%s", *application, *name)
	input := &ssm.DeleteParameterInput{
		Name: &ssmparamname,
	}

	_, err := c.svc.DeleteParameter(input)
	if err != nil {
		panic(fmt.Sprintf("Error authorizing: %s\n", err.Error()))
	}
	return nil
}
