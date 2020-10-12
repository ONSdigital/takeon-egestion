package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/aws/aws-sdk-go/service/sqs"
)

var region = os.Getenv("AWS_REGION")
var queueName = "db-export-input"

func main() {

	lambda.Start(handle)

}

func createSession() {
	fmt.Printf("Region: %q\n", region)
	config := &aws.Config{
		Region: aws.String(region),
	}

	sess := session.New(config)

	return sess
}


func handle() {
	fmt.Println("Starting the application...")\

	session := createSession()
	go sendToSqs(session)

	cdbExport := make(chan string)
	go callGraphqlEndpoint(cdbExport)
	var wg sync.WaitGroup
	wg.Add(1)
	go saveToS3(cdbExport, &wg)
	wg.Wait()
}

func callGraphqlEndpoint(cdbExport chan string) {
	var gqlEndpoint = os.Getenv("GRAPHQL_ENDPOINT")
	fmt.Println("Going to access  Graphql Endpoint: ", gqlEndpoint)
	response, err := http.Get(gqlEndpoint)
	if err != nil {
		fmt.Printf("The HTTP request failed with error %s\n", err)
	} else {
		data, _ := ioutil.ReadAll(response.Body)
		cdbExport <- string(data)
		fmt.Println("Accessing Graphql Endpoint done")
	}
}

func saveToS3(cdbExport chan string, waitGroup *sync.WaitGroup) {

	dbExport := <-cdbExport
	var bucketFilenamePrefix = "takeon-data-export-"

	currentTime := time.Now().Format("2006-01-02-15:04:05")

	// fmt.Printf("Region: %q\n", region)
	// config := &aws.Config{
	// 	Region: aws.String(region),
	// }

	sess := session.New(config)

	uploader := s3manager.NewUploader(sess)

	bucket := os.Getenv("S3_BUCKET")
	filename := strings.Join([]string{bucketFilenamePrefix, currentTime}, "")
	fmt.Printf("Bucket filename: %q\n", filename)

	reader := strings.NewReader(string(dbExport))
	var err error
	_, err = uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(filename),
		Body:   reader,
	})

	if err != nil {
		fmt.Printf("Unable to upload %q to %q, %v", filename, bucket, err)
	}

	fmt.Printf("Successfully uploaded %q to s3 bucket %q\n", filename, bucket)
	waitGroup.Done()

}


func sendToSqs(sess) {
	
	svc := sqs.New(sess)

	urlResult, err := svc.GetQueueUrl(&sqs.GetQueueUrlInput{
		QueueName: queue,
	})

	queueURL := urlResult.QueueUrl
	int timeout := 5

	msgResult, err := svc.ReceiveMessage(&sqs.ReceiveMessageInput{
		AttributeNames: []*string{
			aws.String(sqs.MessageSystemAttributeNameSentTimestamp),
		},
		MessageAttributeNames: []*string{
			aws.String(sqs.QueueAttributeNameAll),
		},
		QueueUrl:            queueURL,
		MaxNumberOfMessages: aws.Int64(1),
		VisibilityTimeout:   timeout,
	})

	fmt.Println("Message Handle: " + *msgResult.Messages[0].ReceiptHandle)

 }

 