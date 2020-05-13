package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	awsCredentials "github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/ec2rolecreds"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	awsSession "github.com/aws/aws-sdk-go/aws/session"

	"github.com/hashicorp/hcl"
)

type config struct {
	AWSCredentials *awsCredentials.Credentials
	AWSRegion      string
	EC2InstanceID  string
	LogGroupName   string
	LogStreamName  string
	LogPriority    priorityType
	LogUnit        string
	StateFilename  string
	JournalDir     string
	BufferSize     int
}

type fileConfig struct {
	AWSRegion     string `hcl:"aws_region"`
	EC2InstanceID string `hcl:"ec2_instance_id"`
	LogGroupName  string `hcl:"log_group"`
	LogStreamName string `hcl:"log_stream"`
	LogPriority   string `hcl:"log_priority"`
	LogUnit       string `hcl:"log_unit"`
	StateFilename string `hcl:"state_file"`
	JournalDir    string `hcl:"journal_dir"`
	BufferSize    int    `hcl:"buffer_size"`
}

func getLogLevel(priority string) (priorityType, error) {

	logLevels := map[priorityType][]string{
		emergencyP: {"0", "emerg"},
		alertP:     {"1", "alert"},
		criticalP:  {"2", "crit"},
		errorP:     {"3", "err"},
		warningP:   {"4", "warning"},
		noticeP:    {"5", "notice"},
		infoP:      {"6", "info"},
		debugP:     {"7", "debug"},
	}

	for i, s := range logLevels {
		if s[0] == priority || s[1] == priority {
			return i, nil
		}
	}

	return debugP, fmt.Errorf("'%s' is unsupported log priority", priority)
}

func loadConfig(filename string) (*config, error) {
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

	expandFileConfig(&fConfig, metaClient)

	config := &config{}

	if fConfig.AWSRegion != "" {
		config.AWSRegion = fConfig.AWSRegion
	} else {
		region, err := metaClient.Region()
		if err != nil {
			return nil, fmt.Errorf("unable to detect AWS region: %s", err)
		}
		config.AWSRegion = region
	}

	if fConfig.EC2InstanceID != "" {
		config.EC2InstanceID = fConfig.EC2InstanceID
	} else {
		instanceID, err := metaClient.GetMetadata("instance-id")
		if err != nil {
			return nil, fmt.Errorf("unable to detect EC2 instance id: %s", err)
		}
		config.EC2InstanceID = instanceID
	}

	if fConfig.LogPriority == "" {
		// Log everything
		config.LogPriority = debugP
	} else {
		config.LogPriority, err = getLogLevel(fConfig.LogPriority)
		if err != nil {
			return nil, fmt.Errorf("The provided log filtering '%s' is unsupported by systemd", fConfig.LogPriority)
		}
	}

	config.LogUnit = fConfig.LogUnit
	config.LogGroupName = fConfig.LogGroupName

	if fConfig.LogStreamName != "" {
		config.LogStreamName = fConfig.LogStreamName
	} else {
		// By default we use the instance id as the stream name.
		config.LogStreamName = config.EC2InstanceID
	}

	config.StateFilename = fConfig.StateFilename
	config.JournalDir = fConfig.JournalDir

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

func (c *config) newAWSSession() *awsSession.Session {
	config := &aws.Config{
		Credentials: c.AWSCredentials,
		Region:      aws.String(c.AWSRegion),
		MaxRetries:  aws.Int(3),
	}
	return awsSession.New(config)
}

/*
 * Expand variables of the form $Foo or ${Foo} in the user provided config
 * from the EC2Metadata Instance Identity Document
 * [ https://docs.aws.amazon.com/sdk-for-go/api/aws/ec2metadata/#EC2InstanceIdentityDocument ]
 * or the environment
 */
func expandFileConfig(config *fileConfig, metaClient *ec2metadata.EC2Metadata) {
	vars := make(map[string]string)

	// If we can fetch the InstanceIdentityDocument then iterate over the
	// struct extracting the string fields and their values into the vars map
	data, err := metaClient.GetInstanceIdentityDocument()
	if err == nil {
		metadata := reflect.ValueOf(data)

		for i := 0; i < metadata.NumField(); i++ {
			field := metadata.Field(i)
			ftype := metadata.Type().Field(i)
			if field.Type() != reflect.TypeOf("") {
				continue
			}
			vars[ftype.Name] = fmt.Sprintf("%v", field.Interface())
		}
	}

	// Iterate over all the string fields in the fileConfig struct performing
	// Variable expansion on them, with EC2 Instance Identity fields overriding
	// the OS environment
	rconfig := reflect.ValueOf(config)
	for i := 0; i < rconfig.Elem().NumField(); i++ {
		field := rconfig.Elem().Field(i)
		if field.Type() != reflect.TypeOf("") {
			continue
		}
		val := field.Interface().(string)
		if val != "" {
			field.SetString(
				expandBraceVars(
					val,
					func(varname string) string {
						if strings.HasPrefix(varname, "instance.") {
							if val, exists := vars[strings.TrimPrefix(varname, "instance.")]; exists {
								return val
							}
							// Unknown key => empty string
							return ""
						} else if strings.HasPrefix(varname, "env.") {
							return os.Getenv(strings.TrimPrefix(varname, "env."))
						} else {
							// Unknown prefix => empty string
							return ""
						}
					},
				),
			)
		}
	}
}

// Modified version of os.Expand() that only expands ${name} and not $name
func expandBraceVars(s string, mapping func(string) string) string {
	buf := make([]byte, 0, 2*len(s))
	// ${} is all ASCII, so bytes are fine for this operation.
	i := 0
	for j := 0; j < len(s); j++ {
		if s[j] == '$' && j+3 < len(s) && s[j+1] == '{' {
			buf = append(buf, s[i:j]...)
			idx := strings.Index(s[j+2:], "}")
			if idx >= 0 {
				// We have a full ${name} string
				buf = append(buf, mapping(s[j+2:j+2+idx])...)
				j += 2 + idx
			} else {
				// We ran out of string (unclosed ${)
				return string(buf)
			}
			i = j + 1
		}
	}
	return string(buf) + s[i:]
}
