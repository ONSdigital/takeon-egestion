package main

import (
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/ONSDigital/spp-logger/go/spp_logger"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/google/uuid"
)

var region = os.Getenv("AWS_REGION")
var bucket = os.Getenv("S3_BUCKET")
var logLevel = os.Getenv("LOG_LEVEL")
var lambdaName = os.Getenv("AWS_LAMBDA_FUNCTION_NAME")
var loggerConfig spp_logger.Config
var logger *spp_logger.Logger
var logCorrelationID = uuid.NewString()
var mainContext spp_logger.Context
var ContextLoglevel string

// SurveyPeriods arrays in JSON message
type SurveyPeriods struct {
	Survey string `json:"survey"`
	Period string `json:"period"`
}

// InputJSON contains snapshot_id and array of surveyperiod combinations
type InputJSON struct {
	SnapshotID    string          `json:"snapshot_id"`
	SurveyPeriods []SurveyPeriods `json:"surveyperiods"`
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
	env := strings.Split(lambdaName, "-")

	loggerConfig = spp_logger.Config{
		Service:     "Snapshot",
		Component:   lambdaName,
		Environment: env[4],
		Deployment:  env[4],
		Timezone:    "UTC",
	}

	FullLogInitialaisation()

	logger.Info("Application starting...")

	if len(sqsEvent.Records) == 0 {
		logger.Error("An error occured, no SQS message passed to function")
		return errors.New("no sqs message passed to function")
	}

	for _, message := range sqsEvent.Records {
		logger.Debug("The message " + message.MessageId + " for event source " + message.EventSource + " is " + message.Body)
		queueMessage := message.Body
		messageDetails := []byte(queueMessage)
		var messageJSON InputJSON
		parseError := json.Unmarshal(messageDetails, &messageJSON)
		if parseError != nil {
			sendToSqs("", "null", false)
			logger.Error("Error with JSON from inpput queue " + parseError.Error())
			return errors.New("Error with JSON from input queue" + parseError.Error())
		}
		inputMessage, validateError := validateInputMessage(messageJSON)
		if validateError != nil {
			logger.Error("Error with message from input queue")
			return errors.New("error with message from input queue")
		}
		snapshotID := inputMessage.SnapshotID
		var filename, err = getFileName(snapshotID, messageJSON.SurveyPeriods)
		if err != nil {
			logger.Error("Unable to create filename. Invalid survey Period")
			return errors.New("unable to create filename. Invalid Survey Period")
		}
		data, dataError := callGraphqlEndpoint(queueMessage, snapshotID, filename)
		if dataError != nil {
			sendToSqs(snapshotID, "null", false)
			logger.Error("Problem with call to Business Layer")
			return errors.New("problem with call to Business Layer")
		}
		saveToS3(data, filename)
	}
	return nil
}

func getFileName(snapshotID string, surveyPeriods []SurveyPeriods) (string, error) {
	var combinedSurveyPeriods = ""
	var join = ""
	var filename = ""

	if len(surveyPeriods) == 0 {
		return filename, errors.New("survey Period Invalid")
	}
	for _, item := range surveyPeriods {
		combinedSurveyPeriods = combinedSurveyPeriods + join + item.Survey + "_" + item.Period
		join = "-"
	}
	var bucketFilenamePrefix = "snapshot"
	filename = strings.Join([]string{bucketFilenamePrefix, combinedSurveyPeriods, snapshotID}, "-")
	return filename, nil
}

func callGraphqlEndpoint(message string, snapshotID string, filename string) (string, error) {
	var gqlEndpoint = os.Getenv("GRAPHQL_ENDPOINT")
	logger.Info("Accessing GraphQL endpoint: ", gqlEndpoint)
	response, err := http.Post(gqlEndpoint, "application/json; charset=UTF-8", strings.NewReader(message))
	logger.Debug("Sending message to business layer: ", message)
	if err != nil {
		logger.Error("The HTTP request failed with error: ", err)
		sendToSqs(snapshotID, "null", false)
	} else {
		data, _ := ioutil.ReadAll(response.Body)
		cdbExport := string(data)
		logger.Debug("Data from BL after successful call: " + cdbExport)
		if strings.Contains(cdbExport, "Error loading data for db Export") {
			logger.Error("Error with business Layer")
			return "", errors.New("error with Business Layer")
		}
		logger.Info("Accessing Graphql endpoint done")
		sendToSqs(snapshotID, filename, true)
		return cdbExport, nil
	}
	return "", nil
}

func saveToS3(dbExport string, filename string) {

	config := &aws.Config{
		Region: aws.String(region),
	}

	sess := session.New(config)

	uploader := s3manager.NewUploader(sess)

	reader := strings.NewReader(string(dbExport))
	var err error
	_, err = uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(filename),
		Body:   reader,
	})

	if err != nil {

		logger.Error("Unable to upload "+filename+" to "+bucket+" with error: ", err)
	}

	logger.Info("Successfully uploaded "+filename+" to s3 bucket ", bucket)

}

func sendToSqs(snapshotid string, filename string, successful bool) {

	queue := aws.String(os.Getenv("DB_EXPORT_OUTPUT_QUEUE"))

	config := &aws.Config{
		Region: aws.String(region),
	}

	sess := session.New(config)

	svc := sqs.New(sess)

	urlResult, err := svc.GetQueueUrl(&sqs.GetQueueUrlInput{
		QueueName: queue,
	})

	if err != nil {
		logger.Error("Unable to find DB Export input queue. ", *queue, err)
	}

	location := "s3://" + bucket + "/" + filename

	queueURL := urlResult.QueueUrl

	outputMessage := &OutputMessage{
		SnapshotID: snapshotid,
		Location:   location,
		Successful: successful,
	}

	DataToSend, err := json.Marshal(outputMessage)
	if err != nil {
		logger.Error("An error occured while marshaling DataToSend: ", err)
	}
	logger.Info("DataToSend: ", string(DataToSend))

	_, error := svc.SendMessage(&sqs.SendMessageInput{
		MessageBody: aws.String(string(DataToSend)),
		QueueUrl:    queueURL,
	})

	if error != nil {
		logger.Error("Unable to send to DB Export output queue ", *queue, error)
	}

}

func validateInputMessage(messageJSON InputJSON) (InputJSON, error) {
	if messageJSON.SnapshotID == "" {
		sendToSqs("", "null", false)
		return messageJSON, errors.New("no SnapshotID given in message")
	} else if len(messageJSON.SurveyPeriods) == 0 {
		sendToSqs(messageJSON.SnapshotID, "null", false)
		return messageJSON, errors.New("no Survey/period combinations given in message")
	}
	return messageJSON, nil
}

// This function initialises a complete logger context including survey, reference and period fields
func FullLogInitialaisation() {

	mainContext = map[string]string{
		"log_level":            logLevel,
		"log_correlation_id":   logCorrelationID,
		"log_correlation_type": "Snapshot"}

	logger, _ = spp_logger.NewLogger(loggerConfig, mainContext, "", os.Stdout)
}
