package dblock

import (
	"context"
	"hash/fnv"
	"log/slog"
	"os"
	"sync"
	"time"
)

type Locker struct {
	querier    Querier
	syncMap    sync.Map
	dialect    Dialect
	logger     Logger
	defaultTtl time.Duration
}

type LockerOption func(*Locker)

// WithLogger sets the logger for the locker.
func WithLogger(logger Logger) LockerOption {
	return func(i *Locker) {
		i.logger = logger
	}
}

// WithDefaultTtl sets the default ttl for the locker.
func WithDefaultTtl(ttl time.Duration) LockerOption {
	return func(instance *Locker) {
		instance.defaultTtl = ttl
	}
}

// New Create a new locker with provided database connection and dialect.
// The Locker is build with simplicity and proficient in mind, it is designed to be used when you have well established persistent database
// which you want to leverage for simple locking use cases.
//
// For more complex use cases, consider using a dedicated distributed system with proper consensus mechanism like zookeeper, etcd...
//
// A locker should be shared across the application with a single database connection for it to work correctly.
//
// Available options: WithLogger, WithDefaultTtl
func New(querier Querier, dialect Dialect, opts ...LockerOption) *Locker {
	l := &Locker{
		querier: querier,
		dialect: dialect,
	}
	for _, opt := range opts {
		opt(l)
	}
	if l.logger == nil {
		l.logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))
	}
	return l
}

// defaultKeyFunc is the default keyFunc used by TryLock.
var defaultKeyFunc = func(key string) any {
	hasher := fnv.New32a()
	// sum32a.Write returns nil error
	_, _ = hasher.Write([]byte(key))
	return hasher.Sum32()
}

// TryLock tries to acquire a lock on the key. If the lock is already acquired, it returns ErrLockAlreadyAcquired.
//
// If ttl equals 0 and the locker's default ttl is 0, the lock is not released automatically unless the database session ends,
// and the caller is responsible for releasing the lock.
//
// Due to the limitation of popular databases lock functions, bigint with postgres and 64 characters with mysql,
// the provided key is hashed using FNV-1a before persisting, this behavior can be overridden by providing a custom keyFunc.
//
// Available options: WithKeyFunc
func (l *Locker) TryLock(ctx context.Context, key string, ttl time.Duration, opts ...LockOption) (releaseFunc ReleaseFunc, err error) {
	_, loaded := l.syncMap.LoadOrStore(key, struct{}{})
	if loaded {
		return nil, ErrLockAlreadyAcquired
	}
	defer func() {
		if err != nil {
			l.syncMap.Delete(key)
		}
	}()
	lockOpts := lockOptions{}
	for _, opt := range opts {
		opt(&lockOpts)
	}

	var persistKey any
	if lockOpts.keyFunc != nil {
		persistKey, err = lockOpts.keyFunc(key)
		if err != nil {
			return nil, err
		}
	} else {
		persistKey = defaultKeyFunc(key)
	}

	if err := l.dialect.TryLock(ctx, l.querier, persistKey); err != nil {
		return nil, err
	}
	if ttl == 0 {
		ttl = l.defaultTtl
	}
	var releaseOnce sync.Once
	if ttl > 0 {
		time.AfterFunc(
			ttl, func() {
				releaseOnce.Do(
					func() {
						defer l.syncMap.Delete(key)
						if err := l.releaseLock(ctx, persistKey); err != nil {
							l.logger.Error("failed to release lock", "key", key, "error", err)
						}
					},
				)
			},
		)
	}

	return func() error {
		var scopedErr error
		releaseOnce.Do(
			func() {
				defer l.syncMap.Delete(key)
				scopedErr = l.releaseLock(ctx, persistKey)
			},
		)
		return scopedErr
	}, nil
}

func (l *Locker) releaseLock(ctx context.Context, persistKey any) error {
	err := l.dialect.ReleaseLock(ctx, l.querier, persistKey)
	if err != nil {
		return err
	}
	return nil
}
