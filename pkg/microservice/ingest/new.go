package ingest

import (
	"strings"
	"time"

	"project-srv/pkg/microservice"

	pkghttp "github.com/smap-hcmut/shared-libs/go/httpclient"
	"github.com/smap-hcmut/shared-libs/go/log"
)

// New creates a new ingest microservice client implementation.
func New(l log.Logger, baseURL string, timeoutMS int, internalKey string) microservice.IngestUseCase {
	timeout := time.Duration(timeoutMS) * time.Millisecond
	if timeoutMS <= 0 {
		timeout = 5 * time.Second
	}
	httpCfg := pkghttp.DefaultConfig()
	httpCfg.Timeout = timeout

	return &implUseCase{
		l:           l,
		baseURL:     strings.TrimRight(baseURL, "/"),
		internalKey: internalKey,
		client:      pkghttp.NewClient(httpCfg),
	}
}
