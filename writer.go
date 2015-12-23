package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	awsCredentials "github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/ec2rolecreds"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	awsSession "github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
)

type Writer struct {
	conn              *cloudwatchlogs.CloudWatchLogs
	nextSequenceToken string
}

func NewWriter(firstSeqToken string) (*Writer, error) {
	metaClient := ec2metadata.New(awsSession.New(&aws.Config{}))
	/*region, err := metaClient.Region()
	if err != nil {
		return nil, fmt.Errorf("couldn't determine AWS region: %s", err)
	}*/
	region := "us-west-2"
	creds := awsCredentials.NewChainCredentials([]awsCredentials.Provider{
		&awsCredentials.EnvProvider{},
		&ec2rolecreds.EC2RoleProvider{
			Client: metaClient,
		},
	})
	config := &aws.Config{
		Credentials: creds,
		Region:      aws.String(region),
		MaxRetries:  aws.Int(3),
	}
	sess := awsSession.New(config)
	conn := cloudwatchlogs.New(sess)

	return &Writer{
		conn:              conn,
		nextSequenceToken: firstSeqToken,
	}, nil
}

func (w *Writer) WriteBatch(records []Record) (string, error) {

	events := make([]*cloudwatchlogs.InputLogEvent, 0, len(records))
	for _, record := range records {
		jsonDataBytes, err := json.MarshalIndent(record, "", "  ")
		if err != nil {
			return "", err
		}
		jsonData := string(jsonDataBytes)

		events = append(events, &cloudwatchlogs.InputLogEvent{
			Message:   aws.String(jsonData),
			Timestamp: aws.Int64(time.Now().Unix() * 1000),
		})
	}

	request := &cloudwatchlogs.PutLogEventsInput{
		LogEvents:     events,
		LogGroupName:  aws.String("scratch"),
		LogStreamName: aws.String("testing"),
	}
	if w.nextSequenceToken != "" {
		request.SequenceToken = aws.String(w.nextSequenceToken)
	}
	result, err := w.conn.PutLogEvents(request)
	if err != nil {
		return "", err
	}
	fmt.Println(result)

	w.nextSequenceToken = *result.NextSequenceToken

	return w.nextSequenceToken, nil
}
