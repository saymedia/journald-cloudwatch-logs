package main

type Priority int

var (
	EMERGENCY Priority = 0
	ALERT     Priority = 1
	CRITICAL  Priority = 2
	ERROR     Priority = 3
	WARNING   Priority = 4
	NOTICE    Priority = 5
	INFO      Priority = 6
	DEBUG     Priority = 7
)

var PriorityJSON = map[Priority][]byte{
	EMERGENCY: []byte("\"EMERG\""),
	ALERT:     []byte("\"ALERT\""),
	CRITICAL:  []byte("\"CRITICAL\""),
	ERROR:     []byte("\"ERROR\""),
	WARNING:   []byte("\"WARNING\""),
	NOTICE:    []byte("\"NOTICE\""),
	INFO:      []byte("\"INFO\""),
	DEBUG:     []byte("\"DEBUG\""),
}

type Record struct {
	InstanceId     string       `json:"instanceId,omitempty"`
	TimeNsec       int64        `json:"-"`
	PID            int          `json:"pid" journald:"_PID"`
	UID            int          `json:"uid" journald:"_UID"`
	GID            int          `json:"gid" journald:"_GID"`
	Command        string       `json:"cmdName,omitempty" journald:"_COMM"`
	Executable     string       `json:"exe,omitempty" journald:"_EXE"`
	CommandLine    string       `json:"cmdLine,omitempty" journald:"_CMDLINE"`
	SystemdUnit    string       `json:"systemdUnit,omitempty" journald:"_SYSTEMD_UNIT"`
	BootId         string       `json:"bootId,omitempty" journald:"_BOOT_ID"`
	MachineId      string       `json:"machineId,omitempty" journald:"_MACHINE_ID"`
	Hostname       string       `json:"hostname,omitempty" journald:"_HOSTNAME"`
	Transport      string       `json:"transport,omitempty" journald:"_TRANSPORT"`
	Priority       Priority     `json:"priority" journald:"PRIORITY"`
	Message        string       `json:"message" journald:"MESSAGE"`
	MessageId      string       `json:"messageId,omitempty" journald:"MESSAGE_ID"`
	Errno          int          `json:"machineId,omitempty" journald:"ERRNO"`
	Syslog         RecordSyslog `json:"syslog,omitempty"`
	Kernel         RecordKernel `json:"kernel,omitempty"`
	Container_Name string       `json:"containerName,omitempty" journald:"CONTAINER_NAME"`
	Container_Tag  string       `json:"containerTag,omitempty" journald:"CONTAINER_TAG"`
	Container_ID   string       `json:"containerID,omitempty" journald:"CONTAINER_ID"`
}

type RecordSyslog struct {
	Facility   int    `json:"facility,omitempty" journald:"SYSLOG_FACILITY"`
	Identifier string `json:"ident,omitempty" journald:"SYSLOG_IDENTIFIER"`
	PID        int    `json:"pid,omitempty" journald:"SYSLOG_PID"`
}

type RecordKernel struct {
	Device    string `json:"device,omitempty" journald:"_KERNEL_DEVICE"`
	Subsystem string `json:"subsystem,omitempty" journald:"_KERNEL_SUBSYSTEM"`
	SysName   string `json:"sysName,omitempty" journald:"_UDEV_SYSNAME"`
	DevNode   string `json:"devNode,omitempty" journald:"_UDEV_DEVNODE"`
}

func (p Priority) MarshalJSON() ([]byte, error) {
	return PriorityJSON[p], nil
}
