/*
   Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.

   This file is licensed under the Apache License, Version 2.0 (the "License").
   You may not use this file except in compliance with the License. A copy of
   the License is located at

    http://aws.amazon.com/apache2.0/

   This file is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR
   CONDITIONS OF ANY KIND, either express or implied. See the License for the
   specific language governing permissions and limitations under the License.
*/

package main

import (
	"encoding/json"
	"io/ioutil"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
)

// Config defines a set of configuration values
type Config struct {
	QueueName string `json:"QueueName"`
	Timeout   int64  `json:"Timeout"`
}

// configFile defines the name of the file containing configuration values
var configFileName = "config.json"

// globalConfig contains the configuration values
var globalConfig Config

func populateConfiguration(t *testing.T) error {
	// Get configuration from config.json

	// Get entire file as a JSON string
	content, err := ioutil.ReadFile(configFileName)
	if err != nil {
		return err
	}

	// Convert []byte to string
	text := string(content)

	// Marshall JSON string in text into global struct
	err = json.Unmarshal([]byte(text), &globalConfig)
	if err != nil {
		return err
	}

	if globalConfig.QueueName == "" {
		// Create unique, random queue name
		id := uuid.New()
		globalConfig.QueueName = "myqueue-" + id.String()
	}

	if globalConfig.Timeout < 0 {
		globalConfig.Timeout = 0
	}

	if globalConfig.Timeout > 20 {
		globalConfig.Timeout = 20
	}

	t.Log("Queue name:        " + globalConfig.QueueName)
	t.Log("Timeout (seconds): " + strconv.Itoa(int(globalConfig.Timeout)))

	return nil
}

func createQueue(sess *session.Session, queueName string) (string, error) {
	// Create a SQS service client
	svc := sqs.New(sess)

	result, err := svc.CreateQueue(&sqs.CreateQueueInput{
		QueueName: aws.String(queueName),
		Attributes: map[string]*string{
			"DelaySeconds":           aws.String("60"),
			"MessageRetentionPeriod": aws.String("86400"),
		},
	})
	if err != nil {
		return "", err
	}

	return *result.QueueUrl, nil
}

func sendMessage(sess *session.Session, queueURL string) (string, error) {
	// Create a SQS service client
	svc := sqs.New(sess)

	result, err := svc.SendMessage(&sqs.SendMessageInput{
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
		QueueUrl:    &queueURL,
	})
	if err != nil {
		return "", err
	}

	return *result.MessageId, nil
}

func deleteQueue(sess *session.Session, queueURL string) error {
	// Create a SQS service client
	svc := sqs.New(sess)

	_, err := svc.DeleteQueue(&sqs.DeleteQueueInput{
		QueueUrl: aws.String(queueURL),
	})
	if err != nil {
		return err
	}

	return nil
}

func TestQueue(t *testing.T) {
	thisTime := time.Now()
	nowString := thisTime.Format("20060102150405")
	t.Log("Starting unit test at " + nowString)

	err := populateConfiguration(t)
	if err != nil {
		t.Fatal(err)
	}

	// Create a session using credentials from ~/.aws/credentials
	// and the region from ~/.aws/config
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	url, err := createQueue(sess, globalConfig.QueueName)
	if err != nil {
		t.Fatal(err)
	}

	t.Log("Created queue " + globalConfig.QueueName)

	msgID, err := sendMessage(sess, url)
	if err != nil {
		t.Fatal(err)
	}

	t.Log("Sent message with ID " + msgID + " to queue " + globalConfig.QueueName)

	msgs, err := ReceiveLPMessages(sess, url, globalConfig.Timeout)
	if err != nil {
		t.Fatal(err)
	}

	foundMsg := false

	t.Log("Message IDs:")
	for _, msg := range msgs {
		t.Log("    " + *msg.MessageId)
		if msgID == *msg.MessageId {
			foundMsg = true
			break
		}
	}

	if foundMsg {
		t.Log("Found sent message")
	} else {
		t.Log("Did NOT find message")
	}

	err = deleteQueue(sess, url)
	if err != nil {
		t.Log("You'll have to delete queue " + globalConfig.QueueName + " yourself")
		t.Fatal(err)
	}

	t.Log("Deleted queue " + globalConfig.QueueName)
}