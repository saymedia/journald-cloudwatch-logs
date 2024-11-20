# journald-cloudwatch-logs

This small utility monitors the systemd journal, managed by `journald`, and writes journal entries into
[AWS Cloudwatch Logs](https://aws.amazon.com/cloudwatch/details/#log-monitoring).

This program is an alternative to the AWS-provided logs agent. The official logs agent copies data from
on-disk text log files into Cloudwatch, while this utility reads directly from the systemd journal.

The journal event data is written to Cloudwatch Logs in JSON format, making it amenable to filtering
using the JSON filter syntax. Log records are translated to Cloudwatch JSON events using a
structure like the following:

```js
{
    "instanceId": "i-xxxxxxxx",
    "pid": 12354,
    "uid": 0,
    "gid": 0,
    "cmdName": "cron",
    "exe": "/usr/sbin/cron",
    "cmdLine": "/usr/sbin/CRON -f",
    "systemdUnit": "cron.service",
    "bootId": "fa58079c7a6d12345678b6ebf1234567",
    "hostname": "ip-10-1-0-15",
    "transport": "syslog",
    "priority": "INFO",
    "message": "pam_unix(cron:session): session opened for user root by (uid=0)",
    "syslog": {
        "facility": 10,
        "ident": "CRON",
        "pid": 12354
    },
    "kernel": {}
}
```

The JSON-formatted log events could also be exported into an AWS ElasticSearch instance using the built-in
sync mechanism, to obtain more elaborate filtering and query capabilities.

## Installation

If you have a binary distribution, you just need to drop the executable file somewhere.

This tool assumes that it is running on an EC2 instance.

This tool uses `libsystemd` to access the journal. systemd-based distributions generally ship
with this already installed, but if yours doesn't you must manually install the library somehow before
this tool will work.

## Configuration

This tool uses a small configuration file to set some values that are required for its operation.
Most of the configuration values are optional and have default settings, but a couple are required.

The configuration file uses a syntax like this:

```js
log_group = "my-awesome-app"

// (you'll need to create this directory before starting the program)
state_file = "/var/lib/journald-cloudwatch-logs/state"
```

The following configuration settings are supported:

* `aws_region`: (Optional) The AWS region whose CloudWatch Logs API will be written to. If not provided,
  this defaults to the region where the host EC2 instance is running.

* `ec2_instance_id`: (Optional) The id of the EC2 instance on which the tool is running. There is very
  little reason to set this, since it will be automatically set to the id of the host EC2 instance.

* `journal_dir`: (Optional) Override the directory where the systemd journal can be found. This is
  useful in conjunction with remote log aggregation, to work with journals synced from other systems.
  The default is to use the local system's journal.

* `log_group`: (Required) The name of the cloudwatch log group to write logs into. This log group must
  be created before running the program.

* `log_priority`: (Optional) The highest priority of the log messages to read (on a 0-7 scale). This defaults
    to DEBUG (all messages). This has a behaviour similar to `journalctl -p <priority>`. At the moment, only
    a single value can be specified, not a range. Possible values are: `0,1,2,3,4,5,6,7` or one of the corresponding
    `"emerg", "alert", "crit", "err", "warning", "notice", "info", "debug"`.
    When a single log level is specified, all messages with this log level or a lower (hence more important)
    log level are read and pushed to CloudWatch. For more information about priority levels, look at
    https://www.freedesktop.org/software/systemd/man/journalctl.html

* `log_unit`: (Optional) The `journalctl` unit to filter. By default,
  not filter. Replicates the behaviour of use the command `journalctl
  -u <log_unit>`. Multiple values can be provided, separated by "`,`".

* `log_stream`: (Optional) The name of the cloudwatch log stream to write logs into. This defaults to
  the EC2 instance id. Each running instance of this application (along with any other applications
  writing logs into the same log group) must have a unique `log_stream` value. If the given log stream
  doesn't exist then it will be created before writing the first set of journal events.

* `state_file`: (Required) Path to a location where the program can write, and later read, some
  state it needs to preserve between runs. (The format of this file is an implementation detail.)

* `buffer_size`: (Optional) The size of the local event buffer where journal events will be kept
  in order to write batches of events to the CloudWatch Logs API. The default is 100. A batch of
  new events will be written to CloudWatch Logs every second even if the buffer does not fill, but
  this setting provides a maximum batch size to use when clearing a large backlog of events, e.g.
  from system boot when the program starts for the first time.

Additionally values in the configuration file can contain variable expansions of the form
${instance.<key>} which will be exapnded from the AWS Instance Identity Document or ${env.<name>}
which will be expanded from the operating system environment variables, if a key does not exist
it expands to the empty string.

At the time of writing, in early 2017, the supported InstanceIdentityDocument variables are:

* `${instance.AvailabilityZone}`: The name of the availability zone the instance is running, eg `ap-southeast-2b`
* `${instance.PrivateIP}`: The AWS internal private IP address of the instance, eg `172.1.2.3`
* `${instance.Version}`: The version of the InstanceIdentityDocument definition?, eg `2010-08-31`
* `${instance.Region}`: The name of the region the instance is running in, eg `ap-southeast-2`
* `${instance.InstanceID}`: The instance identifier, eg `i-0123456789abcdef0`
* `${instance.InstanceType}`: The type of the instance, eg `x1.32xlarge`
* `${instance.AccountID}`: The amazon web services account the instance is running under, eg `098765432123`
* `${instance.ImageID}`: The AMI (image) id the instance was launched from, eg `ami-a1b2c3d4`
* `${instance.KernelID}`: The kernel ID used to launch the instance (PV instances only)
* `${instance.RamdiskID}`: The ramdisk ID used to launch the instance (PV instances only)
* `${instance.Architecture}`: The CPU architecture of the instance, eg `x86_64`

### AWS API access

This program requires access to call some of the Cloudwatch API functions. The recommended way to
achieve this is to create an
[IAM Instance Profile](http://docs.aws.amazon.com/IAM/latest/UserGuide/id_roles_use_switch-role-ec2_instance-profiles.html)
that grants your EC2 instance a role that has Cloudwatch API access. The program will automatically
discover and make use of instance profile credentials.

The following IAM policy grants the required access across all log groups in all regions:

```js
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "logs:CreateLogStream",
                "logs:PutLogEvents",
                "logs:DescribeLogStreams"
            ],
            "Resource": [
                "arn:aws:logs:*:*:log-group:*",
                "arn:aws:logs:*:*:log-group:*:log-stream:*"
            ]
        }
    ]
}
```

In more complex environments you may want to restrict further which regions, groups and streams
the instance can write to. You can do this by adjusting the two ARN strings in the `"Resource"` section:

* The first `*` in each string can be replaced with an AWS region name like `us-east-1`
  to grant access only within the given region.
* The `*` after `log-group` in each string can be replaced with a Cloudwatch Logs log group name
  to grant access only to the named group.
* The `*` after `log-stream` in the second string can be replaced with a Cloudwatch Logs log stream
  name to grant access only to the named stream.

Other combinations are possible too. For more information, see
[the reference on ARNs and namespaces](http://docs.aws.amazon.com/general/latest/gr/aws-arns-and-namespaces.html#arn-syntax-cloudwatch-logs).



### Coexisting with the official Cloudwatch Logs agent

This application can run on the same host as the official Cloudwatch Logs agent but care must be taken
to ensure that they each use a different log stream name. Only one process may write into each log
stream.

## Running on System Boot

This program is best used as a persistent service that starts on boot and keeps running until the
system is shut down. If you're using `journald` then you're presumably using systemd; you can create
a systemd unit for this service. For example:

```
[Unit]
Description=journald-cloudwatch-logs
Wants=basic.target
After=basic.target network.target

[Service]
User=nobody
Group=nobody
ExecStart=/usr/local/bin/journald-cloudwatch-logs /usr/local/etc/journald-cloudwatch-logs.conf
KillMode=process
Restart=on-failure
RestartSec=42s
```

This program is designed under the assumption that it will run constantly from some point during
system boot until the system shuts down.

If the service is stopped while the system is running and then later started again, it will
"lose" any journal entries that were written while it wasn't running. However, on the initial
run after each boot it will clear the backlog of logs created during the boot process, so it
is not necessary to run the program particularly early in the boot process unless you wish
to *promptly* capture startup messages.

## Licence

Copyright (c) 2015 Say Media Inc

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
