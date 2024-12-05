# dblock

Zero dependencies, lightweight go library to utilize your database for distributed locking.

Example:

```go
conn, err := sql.Open("postgres", "postgres://user:password@localhost:5432/dblock&sslmode=disable")
if err != nil {
    return err
}
locker := dblock.New(conn, postgres_dialect.New())
releaseFunc, err := locker.TryLock(context.Background(), "key", time.Minute)
if err != nil {
    return err
}
// your critical section here
if err := releaseFunc(); err != nil {
    return err
}
```

### Usage

```go
// New Create a new locker with provided database connection and dialect.
// The Locker is build with simplicity and proficient in mind, it is designed to be used when you have well established persistent database
// which you want to leverage for simple locking use cases.
//
// For more complex use cases, consider using a dedicated distributed system with proper consensus mechanism like zookeeper, etcd...
//
// A locker should be shared across the application with a single database connection for it to work correctly.
//
// Available options: WithLogger, WithDefaultTtl
func New(querier Querier, dialect Dialect, opts ...LockerOption) *Locker
```

```go
// TryLock tries to acquire a lock on the key. If the lock is already acquired, it returns ErrLockAlreadyAcquired.
//
// If ttl equals 0 and the locker's default ttl is 0, the lock is not released automatically unless the database session ends,
// and the caller is responsible for releasing the lock.
//
// Due to the limitation of popular databases lock functions, bigint with postgres and 64 characters with mysql,
// the provided key is hashed using FNV-1a before persisting, this behavior can be overridden by providing a custom keyFunc.
//
// Available options: WithKeyFunc
func (l *Locker) TryLock(ctx context.Context, key string, ttl time.Duration, opts ...LockOption) (releaseFunc ReleaseFunc, err error)
```

### Contributing
Contributions are welcome!