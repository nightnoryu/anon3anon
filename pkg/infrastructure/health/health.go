package health

import (
	"context"
	"net/http"
	"time"

	kitahealth "github.com/nightnoryu/go-kita/health"
	"github.com/nightnoryu/go-kita/log"
)

const pingTimeout = 2 * time.Second

type Pinger interface {
	Ping(ctx context.Context) error
}

func Handler(store Pinger, logger log.Logger) (http.Handler, error) {
	liveness, err := kitahealth.NewLivenessHandler(kitahealth.LivenessConfig{})
	if err != nil {
		return nil, err
	}

	readiness, err := kitahealth.NewReadinessHandler(kitahealth.ReadinessConfig{
		Timeout:      pingTimeout,
		CheckTimeout: pingTimeout,
		Checks: []kitahealth.NamedCheck{
			{Name: "sqlite", Check: store.Ping},
		},
		OnFailure: func(name string, err error) {
			logger.WithFields(log.Fields{"dependency": name}).Error(err, "readiness check failed")
		},
	})
	if err != nil {
		return nil, err
	}

	mux := http.NewServeMux()
	mux.Handle("/healthz", liveness)
	mux.Handle("/readyz", readiness)

	return mux, nil
}
