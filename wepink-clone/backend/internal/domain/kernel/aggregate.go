package kernel

import (
	"time"
)

type DomainEvent interface {
	OccurredOn() time.Time
	EventName() string
}

type BaseDomainEvent struct {
	occurredOn time.Time
}

func NewBaseDomainEvent() BaseDomainEvent {
	return BaseDomainEvent{occurredOn: time.Now()}
}

func (e BaseDomainEvent) OccurredOn() time.Time {
	return e.occurredOn
}

type AggregateRoot struct {
	events []DomainEvent
}

func (a *AggregateRoot) AddEvent(event DomainEvent) {
	a.events = append(a.events, event)
}

func (a *AggregateRoot) Events() []DomainEvent {
	return a.events
}

func (a *AggregateRoot) ClearEvents() {
	a.events = nil
}
