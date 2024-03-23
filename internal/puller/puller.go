package puller

import (
	"context"
	"time"

	"github.com/nikmy/meowbot/pkg/errors"
)

func NewPuller(ps broadcaster, db db) Puller {
	return &puller{
		ps: ps,
		db: db,
	}
}

type puller struct {
	ps broadcaster
	db db
}

func (p *puller) DoWork(ctx context.Context) error {
	toSend, err := p.db.GetReadyAt(ctx, time.Now())
	if err != nil {
		return errors.WrapFail(err, "get reminders to send from repo")
	}

	errs := make([]error, 0, len(toSend))
	for _, r := range toSend {
		err := p.ps.Broadcast(ctx, r.Channels, r)
		if err != nil {
			errs = append(errs, errors.WrapFail(err, "broadcast message"))
		}
	}

	return errors.Join(errs)
}
