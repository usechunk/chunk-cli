package ui

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

type ProgressBar struct {
	total       int64
	current     int64
	description string
	width       int
	writer      io.Writer
	startTime   time.Time
}

func NewProgressBar(total int64, description string) *ProgressBar {
	return &ProgressBar{
		total:       total,
		current:     0,
		description: description,
		width:       40,
		writer:      os.Stdout,
		startTime:   time.Now(),
	}
}

func (p *ProgressBar) Add(n int64) {
	p.current += n
	p.Render()
}

func (p *ProgressBar) Set(current int64) {
	p.current = current
	p.Render()
}

func (p *ProgressBar) Finish() {
	p.current = p.total
	p.Render()
	fmt.Fprintln(p.writer)
}

func (p *ProgressBar) Render() {
	if p.total == 0 {
		return
	}

	percent := float64(p.current) / float64(p.total)
	if percent > 1.0 {
		percent = 1.0
	}

	filled := int(percent * float64(p.width))
	bar := strings.Repeat("█", filled) + strings.Repeat("░", p.width-filled)

	elapsed := time.Since(p.startTime)
	speed := float64(p.current) / elapsed.Seconds()

	var eta string
	if speed > 0 && p.current < p.total {
		remaining := float64(p.total-p.current) / speed
		eta = fmt.Sprintf(" ETA: %s", time.Duration(remaining)*time.Second)
	}

	fmt.Fprintf(p.writer, "\r%s: [%s] %.1f%%%s",
		p.description, bar, percent*100, eta)
}

type Spinner struct {
	message string
	frames  []string
	current int
	writer  io.Writer
	stop    chan bool
	done    chan bool
}

func NewSpinner(message string) *Spinner {
	return &Spinner{
		message: message,
		frames:  []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		current: 0,
		writer:  os.Stdout,
		stop:    make(chan bool),
		done:    make(chan bool),
	}
}

func (s *Spinner) Start() {
	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-s.stop:
				s.done <- true
				return
			case <-ticker.C:
				fmt.Fprintf(s.writer, "\r%s %s", s.frames[s.current], s.message)
				s.current = (s.current + 1) % len(s.frames)
			}
		}
	}()
}

func (s *Spinner) Stop() {
	s.stop <- true
	<-s.done
	fmt.Fprintf(s.writer, "\r")
}

func (s *Spinner) Success(message string) {
	s.Stop()
	fmt.Fprintf(s.writer, "✓ %s\n", message)
}

func (s *Spinner) Error(message string) {
	s.Stop()
	fmt.Fprintf(s.writer, "✗ %s\n", message)
}

func PrintSuccess(message string) {
	fmt.Printf("✓ %s\n", message)
}

func PrintError(message string) {
	fmt.Fprintf(os.Stderr, "✗ %s\n", message)
}

func PrintWarning(message string) {
	fmt.Printf("⚠ %s\n", message)
}

func PrintInfo(message string) {
	fmt.Printf("ℹ %s\n", message)
}
