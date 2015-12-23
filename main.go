package main

import (
	"fmt"
	"os"

	"github.com/coreos/go-systemd/sdjournal"
)

func main() {
	err := run()
	if err != nil {
		os.Stderr.WriteString(err.Error())
		os.Stderr.Write([]byte{'\n'})
		os.Exit(2)
	}
}

func run() error {
	journal, err := sdjournal.NewJournal()
	if err != nil {
		return fmt.Errorf("error opening journal: %s", err)
	}
	defer journal.Close()

	state, err := OpenState("./tmp-state")
	if err != nil {
		return fmt.Errorf("Failed to open state file: %s", err)
	}

	lastBootId, nextSeq := state.LastState()

	writer, err := NewWriter(nextSeq)
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

	bufSize := 100

	records := make(chan *Record)
	batches := make(chan []Record)

	go ReadRecords(journal, records, skip)
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

		fmt.Println("Journal ", batch)
	}

	return nil
}
