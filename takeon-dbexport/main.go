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
var goContext LoggerContextStruct
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

// LoggerContextStruct ...
type LoggerContextStruct struct {
	Log_level            string `json:"log_level"`
	Log_correlation_id   string `json:"log_correlation_id"`
	Log_correlation_type string `json:"log_correlation_type"`
	Survey               string `json:"survey"`
	Period               string `json:"period"`
	Reference            string `json:"reference"`
}

func main() {

	lambda.Start(handle)

}

func handle(ctx context.Context, sqsEvent events.SQSEvent) error {
	config := Config{}
	env := strings.Split(lambdaName, "-")

	loggerConfig = spp_logger.Config{
		Service:     "Validation",
		Component:   lambdaName,
		Environment: env[4],
		Deployment:  env[4],
		Timezone:    "UTC",
	}

	config = sqsEvent.Payload

	FullLogInitialaisation(config)

	logger.Info("Application starting...")

	if len(sqsEvent.Records) == 0 {
		logger.Error("An error occured, no SQS message passed to function")
		return errors.New("No SQS message passed to function")
	}

	for _, message := range sqsEvent.Records {
		fmt.Printf("The message %s for event source %s = %s \n", message.MessageId, message.EventSource, message.Body)
		queueMessage := message.Body
		messageDetails := []byte(queueMessage)
		var messageJSON InputJSON
		parseError := json.Unmarshal(messageDetails, &messageJSON)
		if parseError != nil {
			sendToSqs("", "null", false)
			return errors.New("Error with JSON from input queue" + parseError.Error())
		}
		inputMessage, validateError := validateInputMessage(messageJSON)
		if validateError != nil {
			return errors.New("Error with message from input queue")
		}
		snapshotID := inputMessage.SnapshotID
		var filename, err = getFileName(snapshotID, messageJSON.SurveyPeriods)
		if err != nil {
			return errors.New("Unable to create filename. Invalid Survey Period")
		}
		logger.Info("File Name: ", filename)
		fmt.Println("File Name: ", filename)
		data, dataError := callGraphqlEndpoint(queueMessage, snapshotID, filename)
		if dataError != nil {
			sendToSqs(snapshotID, "null", false)
			return errors.New("Problem with call to Business Layer")
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
		return filename, errors.New("Survey Period Invalid")
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
	// fmt.Println("Going to access  Graphql Endpoint: ", gqlEndpoint)
	response, err := http.Post(gqlEndpoint, "application/json; charset=UTF-8", strings.NewReader(message))
	fmt.Println("Message sending over to BL: ", message)
	if err != nil {
		logger.Error("The HTTP request failed with error: %s\n", err)
		// fmt.Printf("The HTTP request failed with error %s\n", err)
		sendToSqs(snapshotID, "null", false)
	} else {
		data, _ := ioutil.ReadAll(response.Body)
		cdbExport := string(data)
		logger.Info("Data from BL after successful call: " + cdbExport)
		// fmt.Println("Data from BL after successful call: " + cdbExport)
		if strings.Contains(cdbExport, "Error loading data for db Export") {
			logger.Error("Error with business Layer")
			return "", errors.New("Error with Business Layer")
		}
		// fmt.Println("Accessing Graphql Endpoint done")
		sendToSqs(snapshotID, filename, true)
		return cdbExport, nil
	}
	return "", nil
}

func saveToS3(dbExport string, filename string) {

	fmt.Printf("Region: %q\n", region)
	config := &aws.Config{
		Region: aws.String(region),
	}

	sess := session.New(config)

	uploader := s3manager.NewUploader(sess)

	fmt.Printf("Bucket filename: %q\n", filename)

	reader := strings.NewReader(string(dbExport))
	var err error
	_, err = uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(filename),
		Body:   reader,
	})

	if err != nil {

		logger.Error("Unable to upload %q to %q with error: %s", filename, bucket, err)
		// fmt.Printf("Unable to upload %q to %q, %v", filename, bucket, err)
	}

	logger.Info("Successfully uploaded %q to s3 bucket %q\n", filename, bucket)
	// fmt.Printf("Successfully uploaded %q to s3 bucket %q\n", filename, bucket)

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
		logger.Error("Unable to find DB Export input queue. %q, %q", *queue, err)
		// fmt.Printf("Unable to find DB Export input queue %q, %q", *queue, err)
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
		logger.Error("An error occured while marshaling DataToSend: %s ", err)
		// fmt.Printf("An error occured while marshaling DataToSend: %s", err)
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

// This function initialises a complete logger context including survey, reference and period fields
func FullLogInitialisation(config Config) {

	logerContext := config.LoggerContext

	ContextLoglevel = logerContext.Log_level

	if ContextLoglevel == "" {
		mainContext = map[string]string{"log_level": logLevel,
			"log_correlation_id":   logCorrelationID,
			"log_correlation_type": "Validation",
			"survey":               config.Survey,
			"period":               config.Period,
			"reference":            config.Reference}
	} else {
		mainContext = map[string]string{"log_level": logerContext.Log_level,
			"log_correlation_id":   logerContext.Log_correlation_id,
			"log_correlation_type": logerContext.Log_correlation_type,
			"survey":               logerContext.Survey,
			"period":               logerContext.Period,
			"reference":            logerContext.Reference,
		}
		goContext = LoggerContextStruct{
			Log_level:            logerContext.Log_level,
			Log_correlation_id:   logerContext.Log_correlation_id,
			Log_correlation_type: logerContext.Log_correlation_type,
			Survey:               logerContext.Survey,
			Period:               logerContext.Period,
			Reference:            logerContext.Reference,
		}
	}

	logger, _ = spp_logger.NewLogger(loggerConfig, mainContext, "", os.Stdout)
}
