package scraper

import (
	"context"
	"sync"
	"time"

	"go.blockdaemon.com/solana/cluster-manager/internal/discovery"
	"go.uber.org/zap"
)

type Scraper struct {
	prober     *Prober
	discoverer discovery.Discoverer
	rootCtx    context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup

	Log *zap.Logger
}

func NewScraper(prober *Prober, discoverer discovery.Discoverer) *Scraper {
	ctx, cancel := context.WithCancel(context.Background())
	return &Scraper{
		prober:     prober,
		discoverer: discoverer,
		rootCtx:    ctx,
		cancel:     cancel,
		Log:        zap.NewNop(),
	}
}

func (s *Scraper) Start(results chan<- ProbeResult, interval time.Duration) {
	s.wg.Add(1)
	go s.run(results, interval)
}

func (s *Scraper) Close() {
	s.cancel()
	s.wg.Wait()
}

func (s *Scraper) run(results chan<- ProbeResult, interval time.Duration) {
	defer s.wg.Done()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		ctx, cancel := context.WithCancel(s.rootCtx)
		go s.scrape(ctx, results)

		select {
		case <-s.rootCtx.Done():
			cancel()
			return
		case <-ticker.C:
			cancel()
		}
	}
}

func (s *Scraper) scrape(ctx context.Context, results chan<- ProbeResult) {
	targets, err := s.discoverer.DiscoverTargets(ctx)
	if err != nil {
		s.Log.Error("Service discovery failed", zap.Error(err))
		return
	}

	var wg sync.WaitGroup
	wg.Add(len(targets))
	for _, target := range targets {
		go func(target string) {
			defer wg.Done()
			info, err := s.prober.Probe(ctx, target)
			results <- ProbeResult{info, err}
		}(target)
	}
	wg.Wait()
}