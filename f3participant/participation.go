package f3participant

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-f3/manifest"
	"github.com/filecoin-project/venus/venus-shared/types"
	logging "github.com/ipfs/go-log/v2"
	"github.com/jpillora/backoff"
	"golang.org/x/sync/errgroup"
	"golang.org/x/xerrors"
)

var log = logging.Logger("f3participant")

const (
	// maxCheckProgressAttempts defines the maximum number of failed attempts
	// before we abandon the current lease and restart the participation process.
	//
	// The default backoff takes 12 attempts to reach a maximum delay of 1 minute.
	// Allowing for 13 failures results in approximately 2 minutes of backoff since
	// the lease was granted. Given a lease validity of up to 5 instances, this means
	// we would give up on checking the lease during its mid-validity period;
	// typically when we would try to renew the participation ticket. Hence, the value
	// to 13.
	ParticipationCheckProgressMaxAttempts = 13

	// ParticipationLeaseTerm is the number of instances the miner will attempt to lease from nodes.
	ParticipationLeaseTerm = 5
)

type F3ParticipationAPI interface {
	F3GetOrRenewParticipationTicket(ctx context.Context, minerID address.Address, previous types.F3ParticipationTicket, instances uint64) (types.F3ParticipationTicket, error) //perm:sign
	F3Participate(ctx context.Context, ticket types.F3ParticipationTicket) (types.F3ParticipationLease, error)
	F3GetManifest(ctx context.Context) (*manifest.Manifest, error)
}

type Participant struct {
	node                     F3ParticipationAPI
	participant              address.Address
	backoff                  *backoff.Backoff
	maxCheckProgressAttempts int
	leaseTerm                uint64

	runningCtx context.Context
	cancelCtx  context.CancelFunc
	errgrp     *errgroup.Group
}

func NewParticipant(ctx context.Context,
	node F3ParticipationAPI,
	participant address.Address,
	backoff *backoff.Backoff,
	maxCheckProgress int,
	leaseTerm uint64,
) *Participant {
	runningCtx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	errgrp, runningCtx := errgroup.WithContext(runningCtx)
	return &Participant{
		node:                     node,
		participant:              participant,
		backoff:                  backoff,
		maxCheckProgressAttempts: maxCheckProgress,
		leaseTerm:                leaseTerm,
		runningCtx:               runningCtx,
		cancelCtx:                cancel,
		errgrp:                   errgrp,
	}
}

func (p *Participant) Start(ctx context.Context) error {
	p.errgrp.Go(func() error {
		return p.run(p.runningCtx)
	})
	return nil
}

func (p *Participant) Stop(ctx context.Context) error {
	p.cancelCtx()
	return p.errgrp.Wait()
}

func (p *Participant) run(ctx context.Context) (_err error) {
	defer func() {
		if ctx.Err() != nil {
			_err = nil
		}
		if _err != nil {
			_err = fmt.Errorf("F3 participant stopped unexpectedly: %w", _err)
			log.Error(_err)
		}
	}()

	var ticket types.F3ParticipationTicket
	for ctx.Err() == nil {
		var err error
		start := time.Now()
		ticket, err = p.tryGetF3ParticipationTicket(ctx, ticket)
		if err != nil {
			return err
		}
		err = p.tryParticipate(ctx, ticket)
		if err != nil {
			return err
		}
		const minPeriod = 500 * time.Millisecond
		if sinceLastLoop := time.Since(start); sinceLastLoop < minPeriod {
			select {
			case <-time.After(minPeriod - sinceLastLoop):
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		log.Info("Renewing F3 participation")
	}
	return ctx.Err()
}

func (p *Participant) tryGetF3ParticipationTicket(ctx context.Context, previousTicket types.F3ParticipationTicket) (types.F3ParticipationTicket, error) {
	p.backoff.Reset()
	for ctx.Err() == nil {
		switch ticket, err := p.node.F3GetOrRenewParticipationTicket(ctx, p.participant, previousTicket, p.leaseTerm); {
		case ctx.Err() != nil:
			return types.F3ParticipationTicket{}, ctx.Err()
		case err != nil && strings.Contains(err.Error(), types.ErrF3Disabled.Error()):
			log.Errorw("Cannot participate in F3 as it is disabled.", "err", err)
			return types.F3ParticipationTicket{}, xerrors.Errorf("acquiring F3 participation ticket: %w", err)
		case err != nil:
			log.Debugw("Failed to acquire F3 participation ticket; retrying after backoff", "backoff", p.backoff.Duration(), "err", err)
			p.backOff(ctx)
			log.Debugw("Reattempting to acquire F3 participation ticket.", "attempts", p.backoff.Attempt())
			continue
		default:
			log.Debug("Successfully acquired F3 participation ticket")
			return ticket, nil
		}
	}
	return types.F3ParticipationTicket{}, ctx.Err()
}

func (p *Participant) getManifest(ctx context.Context) (*manifest.Manifest, error) {
	p.backoff.Reset()
	for ctx.Err() == nil {
		switch manifest, err := p.node.F3GetManifest(ctx); {
		case err != nil && strings.Contains(err.Error(), types.ErrF3Disabled.Error()):
			log.Errorw("cannot await F3 participation lease expiry as F3 is disabled.", "err", err)
			return nil, fmt.Errorf("awaiting F3 participation lease expiry: %w", err)
		case err != nil:
			log.Errorw("Error when fetching F3 manifest. Retrying after backoff.", "attempts", p.backoff.Attempt(), "backoff", p.backoff.Duration(), "err", err)
		case manifest == nil:
			// Can happen if we reboot and have no manifest.
			log.Warnw("Received no F3 manifest from lotus. Retrying after backoff.", "attempts", p.backoff.Attempt(), "backoff", p.backoff.Duration())
		default:
			return manifest, nil
		}
		p.backOff(ctx)
	}
	return nil, ctx.Err()
}

func (p *Participant) tryParticipate(ctx context.Context, ticket types.F3ParticipationTicket) error {
	p.backoff.Reset()
	renewLeaseWithin := p.leaseTerm / 2
	var (
		manifest  *manifest.Manifest
		haveLease bool
	)
	for ctx.Err() == nil {
		lease, err := p.node.F3Participate(ctx, ticket)
		switch {
		case ctx.Err() != nil:
			return ctx.Err()
		case err != nil && strings.Contains(err.Error(), types.ErrF3Disabled.Error()):
			log.Errorw("Cannot participate in F3 as it is disabled.", "err", err)
			return xerrors.Errorf("attempting F3 participation with ticket: %w", err)
		case err != nil && strings.Contains(err.Error(), types.ErrF3ParticipationTicketExpired.Error()):
			log.Warnw("F3 participation ticket expired while attempting to participate. Acquiring a new ticket.", "attempts", p.backoff.Attempt(), "err", err)
			return nil
		case err != nil && strings.Contains(err.Error(), types.ErrF3ParticipationTicketStartBeforeExisting.Error()):
			log.Warnw("F3 participation ticket starts before the existing lease. Acquiring a new ticket.", "attempts", p.backoff.Attempt(), "err", err)
			return nil
		case err != nil && strings.Contains(err.Error(), types.ErrF3ParticipationTicketInvalid.Error()):
			log.Errorw("F3 participation ticket is not valid. Acquiring a new ticket after backoff.", "backoff", p.backoff.Duration(), "attempts", p.backoff.Attempt(), "err", err)
			p.backOff(ctx)
			return nil
		case err != nil && strings.Contains(err.Error(), types.ErrF3ParticipationIssuerMismatch.Error()):
			log.Warnw("Node is not the issuer of F3 participation ticket. Miner maybe load-balancing or node has changed. Retrying F3 participation after backoff.", "backoff", p.backoff.Duration(), "err", err)
			p.backOff(ctx)
			log.Debugw("Reattempting F3 participation with the same ticket.", "attempts", p.backoff.Attempt())
			continue
		case err != nil && strings.Contains(err.Error(), types.ErrF3NotReady.Error()):
			log.Warnw("F3 is not ready. Retrying F3 participation after backoff.", "backoff", p.backoff.Duration(), "err", err)
			p.backOff(ctx)
			continue
		case err != nil:
			if p.backoff.Attempt() > float64(p.maxCheckProgressAttempts) {
				log.Errorw("too many failures while attempting to check F3  progress. Restarting participation.", "attempts", p.backoff.Attempt(), "err", err)
				return nil
			}
			log.Errorw("Unexpected error while attempting F3 participation. Retrying after backoff", "backoff", p.backoff.Duration(), "attempts", p.backoff.Attempt(), "err", err)
			p.backOff(ctx)
			continue
		case lease.ValidityTerm <= renewLeaseWithin:
			return nil
		default:
			// we succeeded so reset the backoff.
			p.backoff.Reset()
		}

		// Log the first time we give out the lease.
		if !haveLease {
			log.Infow("Successfully acquired F3 participation lease.",
				"issuer", lease.Issuer,
				"not-before", lease.FromInstance,
				"not-after", lease.ToInstance(),
			)
			haveLease = true
		}

		// Fetch the manifest if necessary.
		if manifest == nil || lease.Network != manifest.NetworkName {
			manifest, err = p.getManifest(ctx)
			if err != nil {
				return err
			}
			if manifest.NetworkName != lease.Network {
				log.Warnf("Got a manifest for network %q while waiting for a lease on network %q. Getting another ticket.", manifest.NetworkName, lease.Network)
				return nil
			}
		}

		// Wait until we think we may need to renew the lease.
		waitTime := time.Duration(lease.ValidityTerm-renewLeaseWithin) * manifest.CatchUpAlignment
		if waitTime == 0 {
			waitTime = 100 * time.Millisecond
		}
		log.Debugf("F3 participation lease is valid for further %d instances. Re-checking after %s.", lease.ValidityTerm, waitTime)
		p.backOffFor(ctx, waitTime)
	}
	return ctx.Err()
}

func (p *Participant) backOff(ctx context.Context) {
	p.backOffFor(ctx, p.backoff.Duration())
}

func (p *Participant) backOffFor(ctx context.Context, d time.Duration) {
	// Create a timer every time to avoid potential risk of deadlock or the need for
	// mutex despite the fact that f3Participator is never (and should never) be
	// called from multiple goroutines.
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return
	case <-timer.C:
	}
}
