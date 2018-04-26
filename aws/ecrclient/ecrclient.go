package ecrclient

import (
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/fsouza/go-dockerclient"
)

type Auth struct {
	Token         string
	User          string
	Pass          string
	ProxyEndpoint string
	ExpiresAt     time.Time
}

func AuthenticateECR(region *string, registryId *string) *docker.AuthConfiguration {
	fmt.Printf("Authenticating with ECR, region: %s, registry: %s\n", *region, *registryId)
	sess := session.Must(session.NewSession(&aws.Config{Region: region}))
	svc := ecr.New(sess)

	//	svc := ecr.New(sess, &aws.Config{Region: &region})
	//svc := ecr.New(sess, aws.NewConfig().WithMaxRetries(10))

	//svc := dynamodb.New(&aws.Config{Region: aws.String("us-east-1")})

	// this lets us handle multiple registrieseee
	params := &ecr.GetAuthorizationTokenInput{
		RegistryIds: []*string{
			aws.String(*registryId),
		},
	}

	// request the token
	resp, err := svc.GetAuthorizationToken(params)
	if err != nil {
		panic(fmt.Sprintf("Error authorizing: %s\n", err.Error()))
	}

	//AuthenticateECR

	// fields to send to template
	fields := make([]Auth, len(resp.AuthorizationData))
	for i, auth := range resp.AuthorizationData {

		// extract base64 token
		data, err := base64.StdEncoding.DecodeString(*auth.AuthorizationToken)
		check(err)

		// extract username and password
		token := strings.SplitN(string(data), ":", 2)

		// object to pass to template
		fields[i] = Auth{
			Token:         *auth.AuthorizationToken,
			User:          token[0],
			Pass:          token[1],
			ProxyEndpoint: *(auth.ProxyEndpoint),
			ExpiresAt:     *(auth.ExpiresAt),
		}
	}

	authConfiguration := docker.AuthConfiguration{
		Username:      `AWS`,
		Password:      fields[0].Pass,
		Email:         `none`,
		ServerAddress: fields[0].ProxyEndpoint,
	}

	return &authConfiguration
}

// error handler
func check(e error) {
	if e != nil {
		panic(e.Error())
	}
}
