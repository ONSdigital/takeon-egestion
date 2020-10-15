package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/aws/aws-sdk-go/service/sqs"
)

var region = os.Getenv("AWS_REGION")

// SurveyPeriods arrays in JSON message
type SurveyPeriods struct {
	Survey string
	Period string
}

// InputJSON contains snapshot_id and array of surveyperiod combinations
type InputJSON struct {
	SnapshotID    string `json:"snapshot_id"`
	SurveyPeriods []SurveyPeriods
}

// OutputMessage to send to export-output queue
type OutputMessage struct {
	SnapshotID string `json:"snapshot_id"`
	Location   string `json:"location"`
	Successful bool   `json:"successful"`
}

func main() {

	lambda.Start(handle)

}

func handle(ctx context.Context, sqsEvent events.SQSEvent) error {
	fmt.Println("Starting the application...")

	if len(sqsEvent.Records) == 0 {
		return errors.New("No SQS message passed to function")
	}

	for _, message := range sqsEvent.Records {
		fmt.Printf("The message %s for event source %s = %s \n", message.MessageId, message.EventSource, message.Body)
		queueMessage := message.Body
		messageDetails := []byte(queueMessage)
		var messageJSON InputJSON
		parseError := json.Unmarshal(messageDetails, &messageJSON)
		if parseError != nil {
			fmt.Printf("Error with JSON from db-export-input queue: %v", parseError)
		}
		// if messageJSON.SnapshotID == "" {
		// 	sendToSqs("null", "null", false)
		// 	return errors.New("No Snapshot ID given in message")
		// }
		// snapshotID := messageJSON.SnapshotID
		// if len(messageJSON.SurveyPeriods) == 0 {
		// 	sendToSqs(snapshotID, "null", false)
		// 	return errors.New("No Survey/period combinations given in message")
		// }
		inputMessage, validateError := validateInputMessage(messageJSON)
		if validateError != nil {
			return errors.New("Error with message from input queue")
		}
		snapshotID := inputMessage.SnapshotID
		survey := inputMessage.SurveyPeriods[0].Survey
		period := inputMessage.SurveyPeriods[0].Period
		var bucketFilenamePrefix = "snapshot"
		filename := strings.Join([]string{bucketFilenamePrefix, survey, period, snapshotID}, "-")
		data := callGraphqlEndpoint(queueMessage, snapshotID, filename)
		saveToS3(data, survey, snapshotID, period, filename)
	}
	return nil
}

func callGraphqlEndpoint(message string, snapshotID string, filename string) string {
	var gqlEndpoint = os.Getenv("GRAPHQL_ENDPOINT")
	fmt.Println("Going to access  Graphql Endpoint: ", gqlEndpoint)
	response, err := http.Post(gqlEndpoint, "application/json; charset=UTF-8", strings.NewReader(message))
	fmt.Println("Message sending over to BL: ", message)
	if err != nil {
		fmt.Printf("The HTTP request failed with error %s\n", err)
		sendToSqs(snapshotID, "null", false)
	} else {
		data, _ := ioutil.ReadAll(response.Body)
		cdbExport := string(data)
		fmt.Println("Accessing Graphql Endpoint done")
		fmt.Println("Data from BL: ", cdbExport)
		sendToSqs(snapshotID, filename, true)
		return cdbExport
	}
	return ""
}

func saveToS3(dbExport string, survey string, snapshotID string, period string, filename string) {

	fmt.Printf("Region: %q\n", region)
	config := &aws.Config{
		Region: aws.String(region),
	}

	sess := session.New(config)

	uploader := s3manager.NewUploader(sess)

	bucket := os.Getenv("S3_BUCKET")
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

}

func sendToSqs(snapshotid string, filename string, successful bool) {

	queue := aws.String(os.Getenv("DB_EXPORT_OUTPUT_QUEUE"))

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

	outputMessage := &OutputMessage{
		SnapshotID: snapshotid,
		Location:   filename,
		Successful: successful,
	}

	DataToSend, err := json.Marshal(outputMessage)
	if err != nil {
		fmt.Printf("An error occured while marshaling DataToSend: %s", err)
	}
	fmt.Printf("DataToSend %v\n", string(DataToSend))

	_, error := svc.SendMessage(&sqs.SendMessageInput{
		MessageBody: aws.String(string(DataToSend)),
		QueueUrl:    queueURL,
	})

	if error != nil {
		fmt.Printf("Unable to send to DB Export output queue %q, %q", *queue, error)
	}

}

func validateInputMessage(messageJSON InputJSON) (InputJSON, error) {
	if messageJSON.SnapshotID == "" {
		sendToSqs("", "null", false)
		return messageJSON, errors.New("No SnapshotID given in message")
	} else if len(messageJSON.SurveyPeriods) == 0 {
		sendToSqs(messageJSON.SnapshotID, "null", false)
		return messageJSON, errors.New("No Survey/period combinations given in message")
	}
	return messageJSON, nil
}
