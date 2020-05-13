package main

import (
	"github.com/coreos/go-systemd/sdjournal"
	"strconv"
	"strings"
)

func addLogFilters(journal *sdjournal.Journal, config *config) {

	// Add Priority Filters
	if config.LogPriority < debugP {
		for p := range priorityJSON {
			if p <= config.LogPriority {
				journal.AddMatch("PRIORITY=" + strconv.Itoa(int(p)))
			}
		}
		journal.AddDisjunction()
	}

	// Add unit filter (multiple values possible, separate by ",")
	if config.LogUnit != "" {
		unitsRaw := strings.Split(config.LogUnit, ",")

		for _, unitRaw := range unitsRaw {
			unit := strings.TrimSpace(unitRaw)
			if unit != "" {
				journal.AddMatch("SYSLOG_IDENTIFIER=" + unit)
				journal.AddDisjunction()
			}
		}

	}
}
