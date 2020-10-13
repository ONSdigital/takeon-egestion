package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/aws/aws-sdk-go/service/sqs"
)

var region = os.Getenv("AWS_REGION")

func main() {

	lambda.Start(handle)

}

func handle(ctx context.Context, sqsEvent events.SQSEvent) {
	fmt.Println("Starting the application...")

	for _, message := range sqsEvent.Records {
		fmt.Printf("The message %s for event source %s = %s \n", message.MessageId, message.EventSource, message.Body)
	}

	//go validateSqsMessage(message.Body)

	cdbExport := make(chan string)
	go callGraphqlEndpoint(cdbExport, message.Body)
	var wg sync.WaitGroup
	wg.Add(1)
	go saveToS3(cdbExport, &wg)
	go sendToSqs()
	wg.Wait()
}

func callGraphqlEndpoint(cdbExport chan string, message string) {
	var gqlEndpoint = os.Getenv("GRAPHQL_ENDPOINT")
	fmt.Println("Going to access  Graphql Endpoint: ", gqlEndpoint)
	// response, err := http.Get(gqlEndpoint)
	response, err := http.NewRequest("GET", gqlEndpoint, strings.NewReader(message))
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

	fmt.Printf("Region: %q\n", region)
	config := &aws.Config{
		Region: aws.String(region),
	}

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

func sendToSqs() {

	queue := aws.String("spp-es-takeon-db-export-output")

	fmt.Printf("Region: %q\n", region)
	config := &aws.Config{
		Region: aws.String(region),
	}

	sess := session.New(config)

	svc := sqs.New(sess)

	urlResult, err := svc.GetQueueUrl(&sqs.GetQueueUrlInput{
		QueueName: queue,
	})

	if err != nil {
		fmt.Printf("Unable to find DB Export input queue %q, %q", *queue, err)
	}

	queueURL := urlResult.QueueUrl

	_, error := svc.SendMessage(&sqs.SendMessageInput{
		DelaySeconds: aws.Int64(10),
		MessageAttributes: map[string]*sqs.MessageAttributeValue{
			"Title": &sqs.MessageAttributeValue{
				DataType:    aws.String("String"),
				StringValue: aws.String("The Whistler"),
			},
			"Author": &sqs.MessageAttributeValue{
				DataType:    aws.String("String"),
				StringValue: aws.String("John Grisham"),
			},
			"WeeksOn": &sqs.MessageAttributeValue{
				DataType:    aws.String("Number"),
				StringValue: aws.String("6"),
			},
		},
		MessageBody: aws.String("Information about current NY Times fiction bestseller for week of 12/11/2016."),
		QueueUrl:    queueURL,
	})

	if error != nil {
		fmt.Printf("Unable to send to DB Export output queue %q, %q", *queue, error)
	}

}

// func validateSqsMessage(string message) string {
// 	return ""
// }
