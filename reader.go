package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/coreos/go-systemd/sdjournal"
)

func ReadRecords(instanceId string, journal *sdjournal.Journal, c chan<- Record, skip uint64) {
	record := &Record{}

	termC := MakeTerminateChannel()
	checkTerminate := func() bool {
		select {
		case <-termC:
			close(c)
			return true
		default:
			return false
		}
	}

	for {
		if checkTerminate() {
			return
		}
		err := UnmarshalRecord(journal, record)
		if err != nil {
			c <- synthRecord(
				fmt.Errorf("error unmarshalling record: %s", err),
			)
			continue
		}

		if skip > 0 {
			skip--
		} else {
			record.InstanceId = instanceId
			c <- *record
		}

		for {
			if checkTerminate() {
				return
			}
			seeked, err := journal.Next()
			if err != nil {
				c <- synthRecord(
					fmt.Errorf("error reading from journal: %s", err),
				)
				// It's likely that we didn't actually advance here, so
				// we should wait a bit so we don't spin the CPU at 100%
				// when we run into errors.
				time.Sleep(2 * time.Second)
				continue
			}
			if seeked == 0 {
				// If there's nothing new in the stream then we'll
				// wait for something new to show up.
				// FIXME: We can actually end up waiting up to 2 seconds
				// to gracefully terminate because of this. It'd be nicer
				// to stop waiting if we get a termination signal, but
				// this will do for now.
				journal.Wait(2 * time.Second)
				continue
			}
			break
		}
	}
}

// BatchRecords consumes a channel of individual records and produces
// a channel of slices of record pointers in sizes up to the given
// batch size.
// If records don't show up fast enough, smaller batches will be returned
// each second as long as at least one item is in the buffer.
func BatchRecords(records <-chan Record, batches chan<- []Record, batchSize int) {
	// We have two buffers here so that we can fill one while the
	// caller is working on the other. The caller is therefore
	// guaranteed that the returned slice will remain valid until
	// the next read of the batches channel.
	var bufs [2][]Record
	bufs[0] = make([]Record, batchSize)
	bufs[1] = make([]Record, batchSize)
	var record Record
	var more bool
	currentBuf := 0
	next := 0
	timer := time.NewTimer(time.Second)
	timer.Stop()

	for {
		select {
		case record, more = <-records:
			if !more {
				close(batches)
				return
			}
			bufs[currentBuf][next] = record
			next++
			if next < batchSize {
				// If we've just added our first record then we'll
				// start the batch timer.
				if next == 1 {
					timer.Reset(time.Second)
				}
				// Not enough records yet, so wait again.
				continue
			}
			break
		case <-timer.C:
			break
		}

		timer.Stop()
		if next == 0 {
			continue
		}

		// If we manage to fall out here then either the buffer is fuull
		// or the batch timer expired. Either way it's time for us to
		// emit a batch.
		batches <- bufs[currentBuf][0:next]

		// Switch buffers before we start building the next batch.
		currentBuf = (currentBuf + 1) % 2
		next = 0
	}
}

// synthRecord produces synthetic records to report errors, so that
// we can stream our own errors directly into cloudwatch rather than
// emitting them through journald and risking feedback loops.
func synthRecord(err error) Record {
	return Record{
		Command:  "journald-cloudwatch-logs",
		Priority: ERROR,
		Message:  json.RawMessage(err.Error()),
	}
}
