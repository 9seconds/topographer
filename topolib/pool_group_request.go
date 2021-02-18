package topolib

import (
	"context"
	"fmt"
	"net"
	"sync"

	"github.com/panjf2000/ants/v2"
)

type resolveIPRequest struct {
	ctx           context.Context
	ip            net.IP
	providers     []Provider
	resultChannel chan<- ResolveResult
	wg            *sync.WaitGroup
}

type poolGroupRequest struct {
	ctx           context.Context
	cancel        context.CancelFunc
	resultChannel chan<- ResolveResult
	providers     []Provider
	wg            *sync.WaitGroup
	pool          *ants.PoolWithFunc
}

func (p *poolGroupRequest) Do(ctx context.Context, ip net.IP) error {
	select {
	case <-ctx.Done():
		return ErrContextIsClosed
	case <-p.ctx.Done():
		return ErrContextIsClosed
	default:
	}

	p.wg.Add(1)

	req := &resolveIPRequest{
		ctx:           p.ctx,
		ip:            ip,
		providers:     p.providers,
		resultChannel: p.resultChannel,
		wg:            p.wg,
	}

	if err := p.pool.Invoke(req); err != nil {
		p.wg.Done()
		p.cancel()

		return fmt.Errorf("cannot schedule a task: %w", err)
	}

	return nil
}

func newPoolGroupRequest(ctx context.Context,
	resultChannel chan<- ResolveResult,
	providers []Provider,
	wg *sync.WaitGroup,
	pool *ants.PoolWithFunc) *poolGroupRequest {
	ctx, cancel := context.WithCancel(ctx)

	return &poolGroupRequest{
		ctx:           ctx,
		wg:            wg,
		resultChannel: resultChannel,
		providers:     providers,
		cancel:        cancel,
		pool:          pool,
	}
}
