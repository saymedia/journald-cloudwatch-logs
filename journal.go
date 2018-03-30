package main

import (
	"strconv"

	"github.com/coreos/go-systemd/sdjournal"
)

func AddLogFilters(journal *sdjournal.Journal, config *Config) {

	if unit := config.Unit; unit != "" {
		journal.AddMatch(sdjournal.SD_JOURNAL_FIELD_SYSTEMD_UNIT + "=" + unit)
	}

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
