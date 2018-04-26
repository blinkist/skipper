package main

import (
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	awssession "github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/blinkist/skipper/helpers"
	"github.com/spf13/cobra"
)

const (
	bucketS3                 = "skipper-repository"
	bucketS3Region           = "eu-central-1"
	bucketS3PathBuildsLinux  = "/builds/linux"
	bucketS3PathBuildsDarwin = "/builds/darwin"
)

var skipperCmd = &cobra.Command{
	Use:   "skipper",
	Short: "Everything related to the skipper tool itself",
	Long:  "Everything related to the skipper tool itself",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			fmt.Printf("No service defined")
			os.Exit(1)
		}
	},
}

//
// /builds/
// /builds/darwin/skipper-latest
// /builds/darwin/skipper-YYMMDDHHMM
// /builds/linux/skipper-latest
// /builds/linux/skipper-YYMMDDHHMM

var skipperUploadCmd = &cobra.Command{
	Use:   "upload",
	Short: "Everything related to the skipper tool itself",
	Long:  "Everything related to the skipper tool itself",
	Run: func(cmd *cobra.Command, args []string) {

		sess := awssession.Must(awssession.NewSession(&aws.Config{
			Region: aws.String(bucketS3Region),
		}))

		t := time.Now()
		pwd, _ := helpers.GetPath()

		bucket := bucketS3
		keyNameDarwin := fmt.Sprintf("%s/skipper-%s", bucketS3PathBuildsDarwin, t.Format("200601021504"))
		keyNameDarwinLatest := fmt.Sprintf("%s/skipper-latest", bucketS3PathBuildsDarwin)

		fileDarwin := fmt.Sprintf("%s/%s", *pwd, "skipper-darwin")
		fhDarwin, err := os.Open(fileDarwin)

		s3raw := s3.New(sess)

		keyNameLinux := fmt.Sprintf("%sskipper-%s", bucketS3PathBuildsDarwin, t.Format("200601021504"))
		fileLinux := fmt.Sprintf("%s/%s", *pwd, "skipper-linux")
		keyNameLinuxLatest := fmt.Sprintf("%s/skipper-latest", bucketS3PathBuildsLinux)
		fhLinux, err := os.Open(fileLinux)

		if err != nil {
			fmt.Printf("The to be uploaded file does not exist:\n%s", fileLinux)
			os.Exit(1)
		}

		fmt.Printf("Uploading %s, to %s on bucket %s", fileDarwin, keyNameDarwin, bucketS3)

		upParams := &s3manager.UploadInput{
			Bucket: &bucket,
			Key:    &keyNameDarwin,
			Body:   fhDarwin,
		}

		uploader := s3manager.NewUploader(sess)

		result, err := uploader.Upload(upParams)

		fmt.Printf("result: %s", result)

		if err != nil {
			fmt.Printf("Error uploading %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("source path: %s", fmt.Sprintf("%s%s", bucket, keyNameDarwin))

		_, err = s3raw.CopyObject(&s3.CopyObjectInput{
			Bucket:     aws.String(bucketS3),
			CopySource: aws.String(fmt.Sprintf("%s%s", bucket, keyNameDarwin)),
			Key:        aws.String(keyNameDarwinLatest),
			ACL:        aws.String("public-read"),
		})

		if err != nil {
			fmt.Printf("The to be copied file does not exist====:\n%s", err)
			os.Exit(1)
		}

		fmt.Printf("Uploading %s, to %s on bucket %s", fileLinux, keyNameLinux, bucketS3)

		upParams = &s3manager.UploadInput{
			Bucket: &bucket,
			Key:    &keyNameLinux,
			Body:   fhLinux,
		}
		result, err = uploader.Upload(upParams)

		if err != nil {
			fmt.Println("Error uploading")
			os.Exit(1)
		}

		_, err = s3raw.CopyObject(&s3.CopyObjectInput{
			Bucket:     aws.String(bucketS3),
			CopySource: aws.String(fmt.Sprintf("%s%s", bucket, keyNameLinux)),
			Key:        aws.String(keyNameLinuxLatest),
			ACL:        aws.String("public-read"),
		})

		if err != nil {
			fmt.Printf("The to be copied file does not exist====:\n%s", err)
			os.Exit(1)
		}

		if err != nil {
			fmt.Println(err)
			return
		}
	},
}

// skipper-repository

func init() {
	RootCmd.AddCommand(skipperCmd)
	skipperCmd.AddCommand(skipperUploadCmd)
}
