package k8sleaderelection

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/apache/dubbo-kubernetes/sail/pkg/leaderelection/k8sleaderelection/k8sresourcelock"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
	"k8s.io/utils/clock"
)

type LeaderElector struct {
	config             LeaderElectionConfig
	observedRecord     k8sresourcelock.LeaderElectionRecord
	observedRawRecord  []byte
	observedTime       time.Time
	reportedLeader     string
	clock              clock.Clock
	metrics            leaderMetricsAdapter
	observedRecordLock sync.Mutex
}

const (
	JitterFactor = 1.2
)

func NewLeaderElector(lec LeaderElectionConfig) (*LeaderElector, error) {
	if lec.LeaseDuration <= lec.RenewDeadline {
		return nil, fmt.Errorf("leaseDuration must be greater than renewDeadline")
	}
	if lec.RenewDeadline <= time.Duration(JitterFactor*float64(lec.RetryPeriod)) {
		return nil, fmt.Errorf("renewDeadline must be greater than retryPeriod*JitterFactor")
	}
	if lec.LeaseDuration < 1 {
		return nil, fmt.Errorf("leaseDuration must be greater than zero")
	}
	if lec.RenewDeadline < 1 {
		return nil, fmt.Errorf("renewDeadline must be greater than zero")
	}
	if lec.RetryPeriod < 1 {
		return nil, fmt.Errorf("retryPeriod must be greater than zero")
	}
	if lec.Callbacks.OnStartedLeading == nil {
		return nil, fmt.Errorf("callback OnStartedLeading must not be nil")
	}
	if lec.Callbacks.OnStoppedLeading == nil {
		return nil, fmt.Errorf("callback OnStoppedLeading  must not be nil")
	}

	if lec.Lock == nil {
		return nil, fmt.Errorf("lock must not be nil")
	}
	le := LeaderElector{
		config:  lec,
		clock:   clock.RealClock{},
		metrics: globalMetricsFactory.newLeaderMetrics(),
	}
	le.metrics.leaderOff(le.config.Name)
	return &le, nil
}

type KeyComparisonFunc func(existingKey string) bool

type LeaderElectionConfig struct {
	Lock            k8sresourcelock.Interface
	LeaseDuration   time.Duration
	RenewDeadline   time.Duration
	RetryPeriod     time.Duration
	ReleaseOnCancel bool
	Name            string
	Callbacks       LeaderCallbacks
	KeyComparison   KeyComparisonFunc
}

type LeaderCallbacks struct {
	OnStartedLeading func(context.Context)
	OnStoppedLeading func()
	OnNewLeader      func(identity string)
}

func (le *LeaderElector) Run(ctx context.Context) {
	defer runtime.HandleCrash()
	defer func() {
		le.config.Callbacks.OnStoppedLeading()
	}()

	if !le.acquire(ctx) {
		return // ctx signaled done
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go le.config.Callbacks.OnStartedLeading(ctx)
	le.renew(ctx)
}

func (le *LeaderElector) acquire(ctx context.Context) bool {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	succeeded := false
	desc := le.config.Lock.Describe()
	klog.Infof("attempting to acquire leader lease %v...", desc)

	// Try to acquire immediately first, then poll if needed
	succeeded = le.tryAcquireOrRenew(ctx)
	le.maybeReportTransition()
	if succeeded {
		le.config.Lock.RecordEvent("became leader")
		le.metrics.leaderOn(le.config.Name)
		klog.Infof("successfully acquired lease %v", desc)
		return true
	}

	// If immediate acquisition failed, poll with jitter
	wait.JitterUntil(func() {
		// Check context before attempting to acquire
		select {
		case <-ctx.Done():
			return
		default:
		}

		succeeded = le.tryAcquireOrRenew(ctx)
		le.maybeReportTransition()
		if !succeeded {
			klog.V(4).Infof("failed to acquire lease %v", desc)
			return
		}
		le.config.Lock.RecordEvent("became leader")
		le.metrics.leaderOn(le.config.Name)
		klog.Infof("successfully acquired lease %v", desc)
		cancel()
	}, le.config.RetryPeriod, JitterFactor, true, ctx.Done())
	return succeeded
}

func (le *LeaderElector) renew(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	wait.Until(func() {
		// Check if context is already cancelled before starting renewal
		select {
		case <-ctx.Done():
			return
		default:
		}

		timeoutCtx, timeoutCancel := context.WithTimeout(ctx, le.config.RenewDeadline)
		defer timeoutCancel()

		err := wait.PollUntilContextCancel(timeoutCtx, le.config.RetryPeriod, true, func(pollCtx context.Context) (bool, error) {
			// Check both the poll context and outer context
			select {
			case <-pollCtx.Done():
				return false, pollCtx.Err()
			case <-ctx.Done():
				return false, ctx.Err()
			default:
			}
			return le.tryAcquireOrRenew(pollCtx), nil
		})

		// Check context after polling to avoid unnecessary processing
		select {
		case <-ctx.Done():
			return
		default:
		}

		le.maybeReportTransition()
		desc := le.config.Lock.Describe()
		if err == nil {
			klog.V(5).Infof("successfully renewed lease %v", desc)
			return
		}
		le.config.Lock.RecordEvent("stopped leading")
		le.metrics.leaderOff(le.config.Name)
		klog.Infof("failed to renew lease %v: %v", desc, err)
		cancel()
	}, le.config.RetryPeriod, ctx.Done())

	// if we hold the lease, give it up
	// Note: release() is non-blocking and won't prevent graceful shutdown
	if le.config.ReleaseOnCancel {
		le.release()
	}
}

func (le *LeaderElector) release() bool {
	if !le.IsLeader() {
		return true
	}
	// Use a timeout context to avoid blocking shutdown indefinitely
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	now := metav1.Now()
	leaderElectionRecord := k8sresourcelock.LeaderElectionRecord{
		LeaderTransitions:    le.observedRecord.LeaderTransitions,
		LeaseDurationSeconds: 1,
		RenewTime:            now,
		AcquireTime:          now,
	}

	// Try to release the lock, but don't block shutdown if it fails
	// During shutdown, the lock may have already been acquired by another process,
	// which is expected behavior. The lock will expire naturally.
	err := le.config.Lock.Update(ctx, leaderElectionRecord)
	if err != nil {
		// Log as error to match Istio behavior, but this is expected during shutdown
		// when the ConfigMap was modified by another process or lease was already expired
		klog.Errorf("Failed to release lock: %v", err)
		return false
	}

	le.setObservedRecord(&leaderElectionRecord)
	return true
}

func (le *LeaderElector) setObservedRecord(observedRecord *k8sresourcelock.LeaderElectionRecord) {
	le.observedRecordLock.Lock()
	defer le.observedRecordLock.Unlock()

	le.observedRecord = *observedRecord
	le.observedTime = le.clock.Now()
}

func (le *LeaderElector) tryAcquireOrRenew(ctx context.Context) bool {
	now := metav1.Now()
	leaderElectionRecord := k8sresourcelock.LeaderElectionRecord{
		HolderIdentity:       le.config.Lock.Identity(),
		HolderKey:            le.config.Lock.Key(),
		LeaseDurationSeconds: int(le.config.LeaseDuration / time.Second),
		RenewTime:            now,
		AcquireTime:          now,
	}

	// 1. obtain or create the ElectionRecord
	oldLeaderElectionRecord, oldLeaderElectionRawRecord, err := le.config.Lock.Get(ctx)
	if err != nil {
		if !errors.IsNotFound(err) {
			klog.Errorf("error retrieving resource lock %v: %v", le.config.Lock.Describe(), err)
			return false
		}
		if err = le.config.Lock.Create(ctx, leaderElectionRecord); err != nil {
			klog.Errorf("error initially creating leader election record: %v", err)
			return false
		}

		le.setObservedRecord(&leaderElectionRecord)

		return true
	}

	// 2. Record obtained, check the Identity & Time
	if !bytes.Equal(le.observedRawRecord, oldLeaderElectionRawRecord) {
		le.setObservedRecord(oldLeaderElectionRecord)

		le.observedRawRecord = oldLeaderElectionRawRecord
	}
	if len(oldLeaderElectionRecord.HolderIdentity) > 0 &&
		le.observedTime.Add(le.config.LeaseDuration).After(now.Time) &&
		!le.IsLeader() {
		if le.config.KeyComparison != nil && le.config.KeyComparison(oldLeaderElectionRecord.HolderKey) {
			// Lock is held and not expired, but our key is higher than the existing one.
			// We will pre-empt the existing leader.
			// nolint: lll
			klog.V(4).Infof("lock is held by %v with key %v, but our key (%v) evicts it", oldLeaderElectionRecord.HolderIdentity, oldLeaderElectionRecord.HolderKey, le.config.Lock.Key())
		} else {
			klog.V(4).Infof("lock is held by %v and has not yet expired", oldLeaderElectionRecord.HolderIdentity)
			return false
		}
	}

	// 3. We're going to try to update. The leaderElectionRecord is set to it's default
	// here. Let's correct it before updating.
	if le.IsLeader() {
		leaderElectionRecord.AcquireTime = oldLeaderElectionRecord.AcquireTime
		leaderElectionRecord.LeaderTransitions = oldLeaderElectionRecord.LeaderTransitions
	} else {
		leaderElectionRecord.LeaderTransitions = oldLeaderElectionRecord.LeaderTransitions + 1
	}

	// update the lock itself
	if err = le.config.Lock.Update(ctx, leaderElectionRecord); err != nil {
		klog.Errorf("Failed to update lock: %v", err)
		return false
	}

	le.setObservedRecord(&leaderElectionRecord)
	return true
}

func (le *LeaderElector) maybeReportTransition() {
	if le.observedRecord.HolderIdentity == le.reportedLeader {
		return
	}
	le.reportedLeader = le.observedRecord.HolderIdentity
	if le.config.Callbacks.OnNewLeader != nil {
		go le.config.Callbacks.OnNewLeader(le.reportedLeader)
	}
}

func (le *LeaderElector) IsLeader() bool {
	return le.getObservedRecord().HolderIdentity == le.config.Lock.Identity()
}

func (le *LeaderElector) getObservedRecord() k8sresourcelock.LeaderElectionRecord {
	le.observedRecordLock.Lock()
	defer le.observedRecordLock.Unlock()

	return le.observedRecord
}
