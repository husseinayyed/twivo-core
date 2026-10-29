package breaker

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/sony/gobreaker/v2"
)

// Registry is a collection of circuit breakers for external dependencies.
type Registry struct {
	breakers map[string]*gobreaker.CircuitBreaker[any]
}

// Group is the default process-wide registry used by infrastructure clients.
var Group = NewGroup()

// NewGroup creates a new circuit breaker registry with default settings for the app's external dependencies.
func NewGroup() *Registry {
	group := &Registry{breakers: make(map[string]*gobreaker.CircuitBreaker[any])}
    group.New("Mongo", 60*time.Second, 30*time.Second)
    group.New("Redis", 30*time.Second, 15*time.Second)
    return group
}

// Execute runs fn under the protection of the named circuit breaker.
func (r *Registry) Execute(service string, fn func() (any, error)) (any, error) {
	cb, ok := r.breakers[service]
	if !ok {
		return nil, fmt.Errorf("breaker: no circuit registered for service %q", service)
	}

	result, err := cb.Execute(fn)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// State returns the current state of a service's breaker.
func (r *Registry) State(service string) gobreaker.State {
	cb, ok := r.breakers[service]
	if !ok {
		return gobreaker.StateClosed
	}
	return cb.State()
}

// New registers a circuit breaker with shared defaults.
func (r *Registry) New(name string, interval, timeout time.Duration) {
	r.breakers[name] = gobreaker.NewCircuitBreaker[any](gobreaker.Settings{
		Name:         name,
		MaxRequests:  3,
		Interval:     interval,
		BucketPeriod: 5 * time.Second,
		Timeout:      timeout,
		OnStateChange: func(name string, from, to gobreaker.State) {
			event := log.Info()
			if to == gobreaker.StateOpen {
				event = log.Warn()
			}
			event.
				Str("service", name).
				Str("from", from.String()).
				Str("to", to.String()).
				Msg("circuit state changed")
		},
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= 4
		},
		IsExcluded: func(err error) bool {
			return errors.Is(err, context.Canceled)
		},
	})
}