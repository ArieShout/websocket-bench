package util

import (
	"log"
	"strings"
)

type countRecord struct {
	name    string
	control string
	value   int64
}

// Counter is a thread safe multiple producer single consumer counter.
// It is thread safe only when there is one consumer draining the results.
type Counter struct {
	stats         map[string]int64
	recordChannel chan countRecord
	resultChannel chan map[string]int64
	stopped       bool
}

func NewCounter() *Counter {
	counter := new(Counter)
	counter.stats = make(map[string]int64)
	counter.recordChannel = make(chan countRecord)
	counter.resultChannel = make(chan map[string]int64)

	counter.start()
	return counter
}

// Stat adds a new count record to the counter.
// This is the only method that can be called by the producers from multiple threads / routines.
func (c *Counter) Stat(name string, value int64) {
	if !c.stopped {
		c.recordChannel <- countRecord{name, "", value}
	}
}

// Snapshot taks a new snapshot of the current counter result.
func (c *Counter) Snapshot() map[string]int64 {
	if !c.stopped {
		c.recordChannel <- countRecord{"", "snapshot", 0}
		return <-c.resultChannel
	}
	return nil
}

// Clear reset all counts that start with prefix.
func (c *Counter) Clear(prefix string) {
	if !c.stopped {
		c.recordChannel <- countRecord{prefix, "clear", 0}
	}
}

// Stop closes the counter and it will not accumulate future records.
func (c *Counter) Stop() {
	c.recordChannel <- countRecord{"", "stop", 0}
}

func (c *Counter) start() {
	go func() {
		defer close(c.recordChannel)
		defer close(c.resultChannel)

		for record := range c.recordChannel {
			if record.control != "" {
				switch record.control {
				case "clear":
					original := c.stats
					c.stats = make(map[string]int64)
					if record.name != "" {
						for k, v := range original {
							if !strings.HasPrefix(k, record.name) {
								c.stats[k] = v
							}
						}
					}
				case "stop":
					return
				case "snapshot":
					snapshot := make(map[string]int64)
					for k, v := range c.stats {
						snapshot[k] = v
					}
					c.resultChannel <- snapshot
				default:
					log.Println("Unknown control command: ", record.name)
				}
			} else {
				c.stats[record.name] += record.value
			}
		}
	}()
}
