package exiftool

import (
	"context"
	"errors"
	"sync"
	"time"
)

type ReuseExiftool struct {
	mux      sync.Mutex
	times    time.Duration
	ticker   *time.Ticker
	exiftool *Exiftool
}

func NewReuseExiftool(ctx context.Context, times time.Duration) (*ReuseExiftool, error) {
	e, err := NewExiftool()
	if err != nil {
		return nil, err
	}

	exif := &ReuseExiftool{
		exiftool: e,
		times:    times,
		ticker:   time.NewTicker(times),
	}
	go exif.run(ctx)
	return exif, nil
}

func (r *ReuseExiftool) Scan(file string) (string, error) {
	r.mux.Lock()
	defer r.mux.Unlock()

	if r.ticker == nil {
		return "", errors.New("reuse exiftool stopped")
	}
	r.ticker.Reset(r.times)
	if r.exiftool == nil {
		e, err := NewExiftool()
		if err != nil {
			return "", err
		}
		r.exiftool = e
	}

	return r.exiftool.Scan(file)
}

func (r *ReuseExiftool) run(ctx context.Context) {
	for {
		select {
		case <-r.ticker.C:
			r.mux.Lock()
			if r.exiftool != nil {
				r.exiftool.Close()
				r.exiftool = nil
			}
			r.mux.Unlock()
		case <-ctx.Done():
			r.mux.Lock()
			switch true {
			case r.exiftool != nil:
				r.exiftool.Close()
				r.exiftool = nil
			case r.ticker != nil:
				r.ticker.Stop()
				r.ticker = nil
			}
			r.mux.Unlock()
			return
		}
	}
}
