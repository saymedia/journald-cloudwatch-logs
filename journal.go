package main

import (
	"github.com/coreos/go-systemd/sdjournal"
	"strconv"
)

func AddLogFilters(journal *sdjournal.Journal, config *Config) {

	// Add Priority Filters
	if config.LogPriority < DEBUG {
		for p, _ := range PriorityJSON {
			if p <= config.LogPriority {
				journal.AddMatch("PRIORITY=" + strconv.Itoa(int(p)))
			}
		}
		journal.AddDisjunction()
	}
}
