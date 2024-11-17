package main

import (
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/textract"
	"github.com/spf13/viper"
)

var textractSession *textract.Textract

func init() {

	awsaccesskey := viperEnvVariable("awsaccesskey")
	awssecret := viperEnvVariable("awssecret")
	awstoken := viperEnvVariable("awstoken")
	awsregion := viperEnvVariable("awsregion")
	textractSession = textract.New(session.Must(session.NewSession(&aws.Config{
		Region:      aws.String(awsregion),
		Credentials: credentials.NewStaticCredentials(awsaccesskey, awssecret, awstoken),
	})))
}

func main() {
	filename := os.Args[1]
	print("processing file name is " + filename)

	extractdocumentcontentAsync(filename)
}

func extractdocumentcontentAsync(filename string) {

	// Define the input for the StartDocumentTextDetection API
	input := &textract.StartDocumentTextDetectionInput{
		DocumentLocation: &textract.DocumentLocation{
			S3Object: &textract.S3Object{
				Bucket: aws.String(viperEnvVariable("awss3bucket")), // Replace with your S3 bucket
				Name:   aws.String(filename),                        // Replace with your document's key (file name)
			},
		},
		NotificationChannel: &textract.NotificationChannel{
			RoleArn:     aws.String(viperEnvVariable("awsrolearn")),  // Replace with your IAM Role ARN
			SNSTopicArn: aws.String(viperEnvVariable("awssnstopic")), // Replace with your SNS topic ARN
		},
	}

	// Start the document text detection job
	resp, err := textractSession.StartDocumentTextDetection(input)
	if err != nil {
		fmt.Printf("Failed to start document text detection: %v", err)
	}

	// Output the JobId
	fmt.Printf("Job started with JobId: %s\n", *resp.JobId)

	// Poll until the job is finished
	status := checkJobStatus(*resp.JobId)

	// If the job succeeded, retrieve the results
	if status == string("SUCCEEDED") {
		getTextDetectionResults(*resp.JobId)
	} else {
		fmt.Println("The job did not complete successfully.")
	}

}

func getTextDetectionResults(jobId string) {

	input := &textract.GetDocumentTextDetectionInput{
		JobId: aws.String(jobId),
	}

	// Get the document text detection results
	resp, err := textractSession.GetDocumentTextDetection(input)
	if err != nil {
		log.Fatalf("Failed to retrieve document text detection results: %v", err)
	}

	// Print out the detected text
	for _, block := range resp.Blocks {
		if string(*block.BlockType) == string("LINE") {
			fmt.Printf("%s\n", string(*block.Text))
		}

	}
}

func checkJobStatus(jobId string) string {
	returnVal := ""
	nexttoken := ""
	input := &textract.GetDocumentTextDetectionInput{
		JobId:      aws.String(jobId),
		MaxResults: aws.Int64(1000),
	}
	for returnVal == "" {
		if nexttoken != "" {
			input = &textract.GetDocumentTextDetectionInput{
				JobId:      aws.String(jobId),
				MaxResults: aws.Int64(1000),
				NextToken:  aws.String(nexttoken),
			}
		}

		// Call the GetDocumentTextDetection API to check the job status
		resp, err := textractSession.GetDocumentTextDetection(input)
		if err != nil {
			fmt.Printf("Failed to check document text detection status: %v", err)
		}

		if *resp.JobStatus == "SUCCEEDED" || *resp.JobStatus == "FAILED" {
			fmt.Printf("Job Status info received as : %s\n", *resp.JobStatus)
			returnVal = *resp.JobStatus

		} else {
			fmt.Printf("Job Status seems to be: %s\n", *resp.JobStatus)
			fmt.Printf("Job %v is still in progress. Waiting...", jobId)
			// Wait before checking the status again
			//time.Sleep(5) // Adjust the polling interval as needed
		}

	}

	return returnVal

}

func viperEnvVariable(key string) string {

	// SetConfigFile explicitly defines the path, name and extension of the config file.
	// Viper will use this and not check any of the config paths.
	// .env - It will search for the .env file in the current directory
	viper.SetConfigFile("./.env")

	// Find and read the config file
	err := viper.ReadInConfig()

	if err != nil {
		log.Fatalf("Error while reading config file %s", err)
	}

	// viper.Get() returns an empty interface{}
	// to get the underlying type of the key,
	// we have to do the type assertion, we know the underlying value is string
	// if we type assert to other type it will throw an error
	value, ok := viper.Get(key).(string)

	// If the type is a string then ok will be true
	// ok will make sure the program not break
	if !ok {
		log.Fatalf("Invalid type assertion")
	}

	return value
}
