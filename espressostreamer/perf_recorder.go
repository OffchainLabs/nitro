package espressostreamer

import (
	"time"

	"github.com/ethereum/go-ethereum/log"
)

type PerfRecorder struct {
	startTime time.Time
	endTime   time.Time
}

func NewPerfRecorder() *PerfRecorder {
	return &PerfRecorder{}
}

func (p *PerfRecorder) SetStartTime(time time.Time) {
	p.startTime = time
}

func (p *PerfRecorder) SetEndTime(time time.Time, logMessage string) {
	p.endTime = time
	duration := p.endTime.Sub(p.startTime)
	log.Debug(logMessage, "start time", p.startTime, "end time", p.endTime, "duration", duration)
}
