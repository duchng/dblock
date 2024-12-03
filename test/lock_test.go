package test

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/duchng/dblock"
)

func TestLocker(t *testing.T) {
	dialect := &mockDialect{}
	locker := dblock.New(nil, dialect)
	t.Run(
		"lock success, release by caller", func(t *testing.T) {
			key := 12345
			dialect.registerCall("TryLock", key)(nil)
			dialect.registerCall("ReleaseLock", key)(nil)
			defer dialect.clear()
			releaseFunc, err := locker.TryLock(
				context.Background(), "key", 0, dblock.WithKeyFunc(
					func(_ string) (any, error) {
						return key, nil
					},
				),
			)
			if err != nil {
				t.Errorf("failed to lock: %s", err.Error())
				return
			}
			if err := releaseFunc(); err != nil {
				t.Errorf("failed to unlock: %s", err.Error())
				return
			}
			if !dialect.isAllRegisteredFunctionsCalled() {
				t.Error("some registered functions are not called")
			}
		},
	)

	t.Run(
		"lock success, release by ttl", func(t *testing.T) {
			key := 12345
			dialect.registerCall("TryLock", key)(nil)
			dialect.registerCall("ReleaseLock", key)(nil)
			defer dialect.clear()
			_, err := locker.TryLock(
				context.Background(), "key", 100*time.Millisecond, dblock.WithKeyFunc(
					func(_ string) (any, error) {
						return key, nil
					},
				),
			)
			if err != nil {
				t.Errorf("failed to lock: %s", err.Error())
				return
			}
			time.Sleep(200 * time.Millisecond)
			if !dialect.isAllRegisteredFunctionsCalled() {
				t.Error("some registered functions are not called")
			}
		},
	)

	t.Run(
		"lock success, default key func", func(t *testing.T) {
			key := "key"
			hasher := fnv.New32a()
			_, _ = hasher.Write([]byte(key))
			persistKey := hasher.Sum32()
			dialect.registerCall("TryLock", persistKey)(nil)
			dialect.registerCall("ReleaseLock", persistKey)(nil)
			defer dialect.clear()
			releaseFunc, err := locker.TryLock(context.Background(), "key", 0)
			if err != nil {
				t.Errorf("failed to lock: %s", err.Error())
				return
			}
			if err := releaseFunc(); err != nil {
				t.Errorf("failed to unlock: %s", err.Error())
				return
			}
			if !dialect.isAllRegisteredFunctionsCalled() {
				t.Error("some registered functions are not called")
			}
		},
	)

	t.Run(
		"lock failed", func(t *testing.T) {
			key := 12345
			dialect.registerCall("TryLock", key)(errors.New("some error"))
			defer dialect.clear()
			_, err := locker.TryLock(context.Background(), "key", 0)
			if err == nil {
				t.Error("expected error")
			}
		},
	)
}

type mockDialect struct {
	registeredCalls map[string]registeredOption
	mu              sync.Mutex
}

func (q *mockDialect) TryLock(ctx context.Context, _ dblock.Querier, key any) error {
	return q.call(ctx, "TryLock", key)
}

func (q *mockDialect) isAllRegisteredFunctionsCalled() bool {
	for _, opt := range q.registeredCalls {
		if !opt.called {
			return false
		}
	}
	return true
}

func (q *mockDialect) ReleaseLock(ctx context.Context, _ dblock.Querier, key any) error {
	return q.call(ctx, "ReleaseLock", key)
}

func (q *mockDialect) registerCall(functionName string, key any) func(error) {
	if q.registeredCalls == nil {
		q.registeredCalls = make(map[string]registeredOption)
	}
	return func(err error) {
		q.registeredCalls[functionName] = registeredOption{
			key: key,
			err: err,
		}
	}
}

func (q *mockDialect) clear() {
	q.registeredCalls = make(map[string]registeredOption)
}

func (q *mockDialect) call(_ context.Context, function string, key any) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	registeredCall, ok := q.registeredCalls[function]
	if !ok {
		return errors.New(fmt.Sprintf("unexpected call: %s", function))
	}
	if !reflect.DeepEqual(key, registeredCall.key) {
		return errors.New("arguments does not match")
	}
	registeredCall.called = true
	q.registeredCalls[function] = registeredCall
	return registeredCall.err
}

type registeredOption struct {
	called bool
	key    any
	err    error
}

var _ dblock.Dialect = (*mockDialect)(nil)
