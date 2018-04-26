package kmsclient

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kms"
)

type Kmsclient struct {
	session *session.Session
	svc     *kms.KMS
	creds   *credentials.Credentials
	logger  *log.Logger
}

func New() *Kmsclient {
	sess := session.New()
	svc := kms.New(sess)

	return &Kmsclient{
		svc:     svc,
		session: sess,
	}
}

func (c *Kmsclient) CreateKey(description *string) error {
	var req kms.CreateKeyInput

	req.Description = aws.String(*description)

	_, err := c.svc.CreateKey(&req)

	return err
}

func (c *Kmsclient) CreateAlias(name *string, targetKeyId *string) error {

	req := &kms.CreateAliasInput{
		AliasName:   aws.String(*name),
		TargetKeyId: aws.String(*targetKeyId),
	}
	fmt.Print(req)
	//
	return nil
}

func (c *Kmsclient) FindKmsAliasByName(name string, marker *string) (*kms.AliasListEntry, error) {
	req := kms.ListAliasesInput{
		Limit: aws.Int64(int64(100)),
	}
	if marker != nil {
		req.Marker = marker
	}
	resp, err := c.svc.ListAliases(&req)
	if err != nil {
		return nil, err
	}

	for _, entry := range resp.Aliases {
		if *entry.AliasName == name {
			return entry, nil
		}
	}
	if *resp.Truncated {
		return c.FindKmsAliasByName(name, resp.NextMarker)
	}

	return nil, nil
}

func (c *Kmsclient) FindKmsKeyByDescription(description string, marker *string) (*kms.AliasListEntry, error) {
	req := kms.ListKeysInput{
		Limit: aws.Int64(int64(100)),
	}
	if marker != nil {
		req.Marker = marker
	}
	resp, err := c.svc.ListAliases(&req)
	if err != nil {
		return nil, err
	}

	for _, entry := range resp.Aliases {
		if *entry.AliasName == name {
			return entry, nil
		}
	}
	if *resp.Truncated {
		return c.FindKmsAliasByName(name, resp.NextMarker)
	}

	return nil, nil
}

func (c *Kmsclient) AliasByNameExists(name string) bool {
	entry, _ := c.FindKmsAliasByName(name, nil)
	if entry == nil {
		return false
	}
	return true
}

//req.Tags = tagsFromMapKMS(v.(map[string]interface{}))

/*
GetAliases (){
// Example iterating over at most 3 pages of a ListAliases operation.
//    pageNum := 0
//    err := client.ListAliasesPages(params,
//        func(page *ListAliasesOutput, lastPage bool) bool {
//            pageNum++
//            fmt.Println(page)
//            return pageNum <= 3
//        })
}

func (c *Kmsclient)  GetAllAliases( application *string, ) ([]*Ssmkeypair, error) {
	    err := client.ListAliasesPages(params,
	        func(page *ListAliasesOutput, lastPage bool) bool {
	            pageNum++
	            fmt.Println(page)
	            return pageNum <= 3
	       })
	}

withDecryption := true
  mypath := fmt.Sprintf("/application/%s" , *application )
	params := &ssm.GetParametersByPathInput{
	Path: &mypath,
	WithDecryption: &withDecryption,
	}
	resp, err := c.svc.GetParametersByPath(params)
	if err != nil {
	log.Fatal(err)
	}
	ssmkeypairs := make([]*Ssmkeypair, len(resp.Parameters))

for i := range resp.Parameters {
	name := *resp.Parameters[i].Name
	value := *resp.Parameters[i].Value
	a := Ssmkeypair{
		Key: &name,
		Value: &value,
	}
	ssmkeypairs[i] = &a
}

return ssmkeypairs, nil
}
*/
