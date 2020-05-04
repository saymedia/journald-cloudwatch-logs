package main

type priorityType int

var (
	emergencyP priorityType = 0
	alertP     priorityType = 1
	criticalP  priorityType = 2
	errorP     priorityType = 3
	warningP   priorityType = 4
	noticeP    priorityType = 5
	infoP      priorityType = 6
	debugP     priorityType = 7
)

var priorityJSON = map[priorityType][]byte{
	emergencyP: []byte("\"EMERG\""),
	alertP:     []byte("\"ALERT\""),
	criticalP:  []byte("\"CRITICAL\""),
	errorP:     []byte("\"ERROR\""),
	warningP:   []byte("\"WARNING\""),
	noticeP:    []byte("\"NOTICE\""),
	infoP:      []byte("\"INFO\""),
	debugP:     []byte("\"DEBUG\""),
}

type record struct {
	InstanceID    string       `json:"instanceId,omitempty"`
	TimeUsec      int64        `json:"-"`
	PID           int          `json:"pid" journald:"_PID"`
	UID           int          `json:"uid" journald:"_UID"`
	GID           int          `json:"gid" journald:"_GID"`
	Command       string       `json:"cmdName,omitempty" journald:"_COMM"`
	Executable    string       `json:"exe,omitempty" journald:"_EXE"`
	CommandLine   string       `json:"cmdLine,omitempty" journald:"_CMDLINE"`
	SystemdUnit   string       `json:"systemdUnit,omitempty" journald:"_SYSTEMD_UNIT"`
	BootID        string       `json:"bootId,omitempty" journald:"_BOOT_ID"`
	MachineID     string       `json:"machineId,omitempty" journald:"_MACHINE_ID"`
	Hostname      string       `json:"hostname,omitempty" journald:"_HOSTNAME"`
	Transport     string       `json:"transport,omitempty" journald:"_TRANSPORT"`
	Priority      priorityType `json:"priority" journald:"PRIORITY"`
	Message       string       `json:"message" journald:"MESSAGE"`
	MessageID     string       `json:"messageId,omitempty" journald:"MESSAGE_ID"`
	Errno         int          `json:"machineId,omitempty" journald:"ERRNO"`
	Syslog        recordSyslog `json:"syslog,omitempty"`
	Kernel        recordKernel `json:"kernel,omitempty"`
	ContainerName string       `json:"containerName,omitempty" journald:"CONTAINER_NAME"`
	ContainerTag  string       `json:"containerTag,omitempty" journald:"CONTAINER_TAG"`
	ContainerID   string       `json:"containerID,omitempty" journald:"CONTAINER_ID"`
}

type recordSyslog struct {
	Facility   int    `json:"facility,omitempty" journald:"SYSLOG_FACILITY"`
	Identifier string `json:"ident,omitempty" journald:"SYSLOG_IDENTIFIER"`
	PID        int    `json:"pid,omitempty" journald:"SYSLOG_PID"`
}

type recordKernel struct {
	Device    string `json:"device,omitempty" journald:"_KERNEL_DEVICE"`
	Subsystem string `json:"subsystem,omitempty" journald:"_KERNEL_SUBSYSTEM"`
	SysName   string `json:"sysName,omitempty" journald:"_UDEV_SYSNAME"`
	DevNode   string `json:"devNode,omitempty" journald:"_UDEV_DEVNODE"`
}

func (p priorityType) marshalJSON() ([]byte, error) {
	return priorityJSON[p], nil
}
