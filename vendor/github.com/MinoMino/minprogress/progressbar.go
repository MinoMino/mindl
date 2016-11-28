// A progress bar for arbitrary units, which can also keep track of
// the speed of progress in terms of units/second. Safe for concurrent use.
//
// To keep track of speeds, have each "progress maker" report how many
// units of progress it has done at desired intervals using Report()
// with a unique identifier. If such a progress maker has finished its
// task, have it report it with Done(). For every X amount of reports,
// it will sample the average speed of all the progress makers and include
// it as part of the formatted progress bar string.
package minprogress

import (
	"container/ring"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/MinoMino/minterm"
)

const (
	AutoWidth          = 0
	UnknownTotal       = 0
	defaultReportCount = 50
)

type Unit struct {
	Size int64
	Name string
}

// Data speed units. Useful for transfer speeds.
var DataUnits = []Unit{
	Unit{1000000000000, "TiB"},
	Unit{1000000000, "GiB"},
	Unit{1000000, "MiB"},
	Unit{1000, "KiB"},
	Unit{1, "B"},
}

// Holds info about the speed of progress. Provides
// methods to get info and to report progress.
// Can be used uninitialized, using the default number of
// 50 reports.
type SpeedInfo struct {
	reports          *ring.Ring
	last             time.Time
	reportCount, buf int
	init             bool
}

// Report n amount of progress made since last call.
func (s *SpeedInfo) Report(n int) {
	if s.init && time.Since(s.last).Nanoseconds() == 0 {
		// If the call since last time is too fast, Since() might evaluate
		// to literally 0, so if that's the case, buffer n and wait for the
		// next call.
		s.buf += n
		return
	}

	if s.reports == nil {
		if s.reportCount == 0 {
			s.reportCount = defaultReportCount
		}

		s.reports = ring.New(s.reportCount)
	}

	if s.init {
		s.buf = 0
		s.reports.Value = float64(n) / time.Since(s.last).Seconds()
		s.reports = s.reports.Next()
	} else {
		s.init = true
	}

	s.last = time.Now()
}

// Get the average speed.
func (s *SpeedInfo) Average() float64 {
	sum := 0.0
	i := 0
	s.reports.Do(func(rep interface{}) {
		if rep == nil {
			return
		}

		sum += rep.(float64)
		i++
	})

	if i == 0 {
		return 0
	}

	return sum / float64(i)
}

type ProgressBar struct {
	// The characters used for the empty and full parts of the bar itself.
	// Uses ░ and █ by default, respectively.
	Empty, Full rune
	// If desired, they can be set to words describing what a unit of progress
	// means. For instance "File" and "Files" if the bar represents how many files
	// have been processed.
	Unit, Units string
	// The speed units used for reporting speed. See DataUnits for an example
	// of units for data transfer/processing.
	SpeedUnits []Unit
	// The width of the console and the padding on the left of the bar.
	// The width is AutoWidth by default, which will automatically determine
	// the appropriate size depending on the terminal size.
	Width, Padding int
	// Number of reports that should be stored to calculate the average.
	// The higher it is, the less volatile it is. Default is 50.
	ReportCount, OverallReportCount int
	// How many calls to Report() it should take for it to sample all
	// speed and calculate an overall average speed. Default is 15.
	ReportsPerSample, reports int
	// A map holding speed information for each individual ID.
	speeds map[int]*SpeedInfo
	// The overall speed of all IDs combined.
	overallSpeed float64
	// Mutex for speed stuff.
	m, om          sync.Mutex
	current, total int
}

// Creates a new progress bar starting at 0 units. If total is set
// to UnknownTotal, no bar will be displayed, but it will still display
// the number of units of progress that have been made and whatnot.
func NewProgressBar(total int) *ProgressBar {
	if total < 0 {
		total = 0
	}
	return &ProgressBar{
		Empty:              '░',
		Full:               '█',
		total:              total,
		Padding:            2,
		speeds:             make(map[int]*SpeedInfo),
		ReportCount:        defaultReportCount,
		OverallReportCount: defaultReportCount,
		ReportsPerSample:   25,
	}
}

// Make n amount of units in progress.
func (p *ProgressBar) Progress(n int) int {
	if p.total == UnknownTotal {
		p.current = max(0, p.current+n)
	} else {
		p.current = max(0, min(p.total, p.current+n))
	}
	return p.current
}

// Report how many units of progress have been made since last call
// for that particular UID.
func (p *ProgressBar) Report(uid, n int) {
	p.m.Lock()
	defer p.m.Unlock()
	var si *SpeedInfo
	if _, ok := p.speeds[uid]; !ok {
		si = &SpeedInfo{}
		p.speeds[uid] = si
	} else {
		si = p.speeds[uid]
	}
	si.Report(n)
	p.reports++

	// Sample sum of averages if we need to.
	if p.reports%p.ReportsPerSample == 0 {
		p.om.Lock()
		defer p.om.Unlock()
		p.overallSpeed = 0.0
		for _, si := range p.speeds {
			p.overallSpeed += si.Average()
		}
	}
}

// Report that the progress of a UID is done. This is important
// to call to keep an accurate overall average.
func (p *ProgressBar) Done(uid int) {
	p.m.Lock()
	delete(p.speeds, uid)
	p.m.Unlock()
}

// Returns the average speed of a particular UID. Returns an error
// if and only if the UID doesn't exist.
func (p *ProgressBar) AverageSpeed(uid int) (float64, error) {
	p.m.Lock()
	defer p.m.Unlock()
	if si, ok := p.speeds[uid]; ok {
		return si.Average(), nil
	}

	return 0, fmt.Errorf("Nonexistent UID: %d", uid)
}

// Returns the average cumulative speed and the number of UID used to
// calculate it.
func (p *ProgressBar) AverageOverallSpeed() (avg float64) {
	p.om.Lock()
	avg = p.overallSpeed
	p.om.Unlock()
	return
}

// Just a helper function to avoid doing unecessary additional mutex locks.
func (p *ProgressBar) avgOverallSpeedAndTotal() (avg float64, total int) {
	p.om.Lock()
	avg = p.overallSpeed
	total = len(p.speeds)
	p.om.Unlock()
	return
}

// Gets the whole formatted progress bar.
func (p *ProgressBar) String() string {
	var units, out string
	if p.Unit != "" && p.Units != "" {
		if p.current == 1 {
			units = " " + p.Unit
		} else {
			units = " " + p.Units
		}
	}

	if p.total == UnknownTotal {
		out = fmt.Sprintf("%s%d / ?%s%s",
			strings.Repeat(" ", p.Padding), p.current, units, p.speedFormat())
	} else {
		percentage := int(100 * float64(p.current) / float64(p.total))
		out = fmt.Sprintf("%s%3d%% %s (%d/%d)%s%s",
			strings.Repeat(" ", p.Padding), percentage, p.bar(),
			p.current, p.total, units, p.speedFormat())
	}

	return out
}

func (p *ProgressBar) bar() string {
	ratio := float64(p.current) / float64(p.total)
	width := p.Width
	if width == AutoWidth {
		cols, _, _ := minterm.TerminalSize()
		width = int((float64(cols) / 4) + 0.5)
	}

	fulls := int((float64(width) * ratio) + 0.5)
	return strings.Repeat(string(p.Full), fulls) + strings.Repeat(string(p.Empty), width-fulls)
}

func (p *ProgressBar) speedFormat() string {
	if p.SpeedUnits == nil {
		return ""
	}

	avg, total := p.avgOverallSpeedAndTotal()
	if total == 0 {
		return ""
	}

	unit := DataUnits[len(DataUnits)-1]
	for _, u := range DataUnits {
		if avg > float64(u.Size) {
			unit = u
			break
		}
	}

	return fmt.Sprintf(" [%3.1f %s/s]", avg/float64(unit.Size), unit.Name)
}

func min(x, y int) int {
	if x < y {
		return x
	}

	return y
}

func max(x, y int) int {
	if x > y {
		return x
	}

	return y
}
