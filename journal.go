package main

import (
	"github.com/coreos/go-systemd/sdjournal"
	"strconv"
	"strings"
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

	// Add unit filter (multiple values possible, separate by ",")
	if config.LogUnit != "" {
		units_raw := strings.Split(config.LogUnit, ",")

		for _, unit_raw := range(units) {
			unit := strings.TrimSpace(unit_raw)
			if unit != "" {
				journal.AddMatch("SYSLOG_IDENTIFIER="+unit)
				journal.AddDisjunction()
			}
		}

	}
}
