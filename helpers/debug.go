package helpers

import (
	"strings"
	"time"

	"github.com/onsi/gomega"
)

var DoSanityCheck = true

func RegisterGomega() {
	gomega.RegisterFailHandler(func(message string, callerSkip ...int) {
		if DoSanityCheck {
			panic(message)
		}
	})
}

func SanityCheck(assertion func() bool, messages ...string) {
	if DoSanityCheck && !assertion() {
		if len(messages) == 0 {
			messages = append(messages, "assertion violated")
		}
		panic(strings.Join(messages, " "))
	}
}

type Overhead struct {
	time time.Duration

	lastStartTime *time.Time
}

var OverHeadMonitor = &Overhead{}

func (o *Overhead) Reset() {
	o.time = time.Duration(0)
	o.lastStartTime = nil
}

func (o *Overhead) collect() {
	if o.Paused() {
		return
	}
	now := time.Now()
	o.time += now.Sub(*(o.lastStartTime))
	o.lastStartTime = &now
}

func (o *Overhead) Paused() bool {
	return o.lastStartTime == nil
}

func (o *Overhead) Resume() {
	if o.Paused() {
		now := time.Now()
		o.lastStartTime = &now
	}
}

func (o *Overhead) Pause() {
	if !o.Paused() {
		o.collect()
		o.lastStartTime = nil
	}
}

func (o *Overhead) Time() time.Duration {
	o.collect()
	return o.time
}
