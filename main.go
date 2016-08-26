package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/coreos/go-systemd/sdjournal"
)

var help = flag.Bool("help", false, "set to true to show this help")

func main() {
	flag.Parse()

	if *help {
		usage()
		os.Exit(0)
	}

	configFilename := flag.Arg(0)
	if configFilename == "" {
		usage()
		os.Exit(1)
	}

	err := run(configFilename)
	if err != nil {
		os.Stderr.WriteString(err.Error())
		os.Stderr.Write([]byte{'\n'})
		os.Exit(2)
	}
}

func usage() {
	os.Stderr.WriteString("Usage: journald-cloudwatch-logs <config-file>\n\n")
	flag.PrintDefaults()
	os.Stderr.WriteString("\n")
}

func run(configFilename string) error {
	config, err := LoadConfig(configFilename)
	if err != nil {
		return fmt.Errorf("error reading config: %s", err)
	}

	var journal *sdjournal.Journal
	if config.JournalDir == "" {
		journal, err = sdjournal.NewJournal()
	} else {
		log.Printf("using journal dir: %s", config.JournalDir)
		journal, err = sdjournal.NewJournalFromDir(config.JournalDir)
	}

	if err != nil {
		return fmt.Errorf("error opening journal: %s", err)
	}
	defer journal.Close()

	AddLogFilters(journal, config)

	state, err := OpenState(config.StateFilename)
	if err != nil {
		return fmt.Errorf("Failed to open %s: %s", config.StateFilename, err)
	}

	lastBootId, nextSeq := state.LastState()

	awsSession := config.NewAWSSession()

	writer, err := NewWriter(
		awsSession,
		config.LogGroupName,
		config.LogStreamName,
		nextSeq,
	)
	if err != nil {
		return fmt.Errorf("error initializing writer: %s", err)
	}

	seeked, err := journal.Next()
	if seeked == 0 || err != nil {
		return fmt.Errorf("unable to seek to first item in journal")
	}

	bootId, err := journal.GetData("_BOOT_ID")
	bootId = bootId[9:] // Trim off "_BOOT_ID=" prefix

	// If the boot id has changed since our last run then we'll start from
	// the beginning of the stream, but if we're starting up with the same
	// boot id then we'll seek to the end of the stream to avoid repeating
	// anything. However, we will miss any items that were added while we
	// weren't running.
	skip := uint64(0)
	if bootId == lastBootId {
		// If we're still in the same "boot" as we were last time then
		// we were stopped and started again, so we'll seek to the last
		// item in the log as an approximation of resuming streaming,
		// though we will miss any logs that were added while we were
		// running.
		journal.SeekTail()
		// Skip the last item so our log will resume only when we get
		// the *next item.
		skip = 1
	}

	err = state.SetState(bootId, nextSeq)
	if err != nil {
		return fmt.Errorf("Failed to write state: %s", err)
	}

	bufSize := config.BufferSize

	records := make(chan *Record)
	batches := make(chan []Record)

	go ReadRecords(config.EC2InstanceId, journal, records, skip)
	go BatchRecords(records, batches, bufSize)

	for batch := range batches {

		nextSeq, err = writer.WriteBatch(batch)
		if err != nil {
			return fmt.Errorf("Failed to write to cloudwatch: %s", err)
		}

		err = state.SetState(bootId, nextSeq)
		if err != nil {
			return fmt.Errorf("Failed to write state: %s", err)
		}

	}

	// We fall out here when interrupted by a signal.
	// Last chance to write the state.
	err = state.SetState(bootId, nextSeq)
	if err != nil {
		return fmt.Errorf("Failed to write state on exit: %s", err)
	}

	return nil
}
