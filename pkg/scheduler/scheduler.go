package scheduler

import (
	"errors"
	"sync"
	"time"
)

var (
	ErrEventInPast = errors.New("event datetime is in the past")
	ErrTimeInvalid = errors.New("datetime format is not in RFC3339")
)

// Payload data associated with an event
type Payload struct {
	Name        string
	ContentType string
	Body        []byte
}

// Event which will run on scheduler
type Event struct {
	datetime string
	payloads []Payload
}

func NewEvent(d string, pay []Payload) *Event {
	cpyLoad := make([]Payload, len(pay))
	copy(cpyLoad, pay)

	return &Event{
		datetime: d,
		payloads: cpyLoad,
	}
}

func (e *Event) Date() (time.Time, error) {
	return time.Parse(time.RFC3339, e.datetime)
}

func (e *Event) Payloads() []Payload {
	cpyLoad := make([]Payload, len(e.payloads))
	copy(cpyLoad, e.payloads)

	return cpyLoad
}

// EventHandler delegates
type EventHandler func(*Scheduler, *Event)

// Scheduler ...
type Scheduler struct {
	delegate EventHandler
	stop     chan struct{}
	pending  chan *Event
	wg       *sync.WaitGroup
}

// New instance of scheduler
func New(d EventHandler) *Scheduler {
	return &Scheduler{
		delegate: d,
		stop:     make(chan struct{}),
		pending:  make(chan *Event, 3),
		wg:       &sync.WaitGroup{},
	}
}

// Schedule an event for specific time
func (s *Scheduler) Schedule(e *Event) error {
	date, err := e.Date()
	if err != nil {
		return ErrTimeInvalid
	}

	now := time.Now()
	if date.Unix() <= now.Unix() {
		return ErrEventInPast
	}

	s.wg.Add(1)
	go func(e *Event) {
		now := time.Now()
		target, _ := e.Date()
		waitDuration := target.Sub(now)

		defer s.wg.Done()
		select {
		case <-time.After(waitDuration):
			s.delegate(s, e)
		case <-s.stop:
			s.pending <- e
		}
	}(e)
	return nil
}

// Minutely run an event every
func (s *Scheduler) Minutely(e *Event) *Scheduler {
	return s.EveryInterval(1*time.Minute, e)
}

// Secondly run an event every
func (s *Scheduler) Secondly(e *Event) *Scheduler {
	return s.EveryInterval(1*time.Second, e)
}

// EveryInterval run an event in every interval
func (s *Scheduler) EveryInterval(duration time.Duration, e *Event) *Scheduler {
	ch := make(chan bool, 2)
	kc := make(chan bool, 1)

	go func() {
		<-kc
		ch <- true
	}()
	go func() {
		time.AfterFunc(duration, func() {
			s.delegate(s, e)
			s.EveryInterval(duration, e)
		})
	}()
	return s
}

func (s *Scheduler) Stop() (events []*Event) {
	close(s.stop)
	go func() {
		s.wg.Wait()
		close(s.pending)
	}()
	for pendingEvent := range s.pending {
		events = append(events, pendingEvent)
	}
	return events
}
