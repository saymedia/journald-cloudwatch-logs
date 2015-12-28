package main

import (
	"fmt"
	"io/ioutil"

	"github.com/aws/aws-sdk-go/aws"
	awsCredentials "github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/ec2rolecreds"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	awsSession "github.com/aws/aws-sdk-go/aws/session"

	"github.com/hashicorp/hcl"
)

type Config struct {
	AWSCredentials *awsCredentials.Credentials
	AWSRegion      string
	EC2InstanceId  string
	LogGroupName   string
	LogStreamName  string
	StateFilename  string
	BufferSize     int
}

type fileConfig struct {
	AWSRegion     string `hcl:"aws_region"`
	EC2InstanceId string `hcl:"ec2_instance_id"`
	LogGroupName  string `hcl:"log_group"`
	LogStreamName string `hcl:"log_stream"`
	StateFilename string `hcl:"state_file"`
	BufferSize    int    `hcl:"buffer_size"`
}

func LoadConfig(filename string) (*Config, error) {
	configBytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var fConfig fileConfig
	err = hcl.Decode(&fConfig, string(configBytes))
	if err != nil {
		return nil, err
	}

	if fConfig.LogGroupName == "" {
		return nil, fmt.Errorf("log_group is required")
	}
	if fConfig.StateFilename == "" {
		return nil, fmt.Errorf("state_file is required")
	}

	metaClient := ec2metadata.New(awsSession.New(&aws.Config{}))

	config := &Config{}

	if fConfig.AWSRegion != "" {
		config.AWSRegion = fConfig.AWSRegion
	} else {
		region, err := metaClient.Region()
		if err != nil {
			return nil, fmt.Errorf("unable to detect AWS region: %s", err)
		}
		config.AWSRegion = region
	}

	if fConfig.EC2InstanceId != "" {
		config.EC2InstanceId = fConfig.EC2InstanceId
	} else {
		instanceId, err := metaClient.GetMetadata("instance-id")
		if err != nil {
			return nil, fmt.Errorf("unable to detect EC2 instance id", err)
		}
		config.EC2InstanceId = instanceId
	}

	config.LogGroupName = fConfig.LogGroupName

	if fConfig.LogStreamName != "" {
		config.LogStreamName = fConfig.LogStreamName
	} else {
		// By default we use the instance id as the stream name.
		config.LogStreamName = config.EC2InstanceId
	}

	config.StateFilename = fConfig.StateFilename

	if fConfig.BufferSize != 0 {
		config.BufferSize = fConfig.BufferSize
	} else {
		config.BufferSize = 100
	}

	config.AWSCredentials = awsCredentials.NewChainCredentials([]awsCredentials.Provider{
		&awsCredentials.EnvProvider{},
		&ec2rolecreds.EC2RoleProvider{
			Client: metaClient,
		},
	})

	return config, nil
}

func (c *Config) NewAWSSession() *awsSession.Session {
	config := &aws.Config{
		Credentials: c.AWSCredentials,
		Region:      aws.String(c.AWSRegion),
		MaxRetries:  aws.Int(3),
	}
	return awsSession.New(config)
}

