package domain

import (
	"context"
	"errors"
	"testing"
	"time"

	goredis "github.com/redis/go-redis/v9"
	"github.com/smap-hcmut/shared-libs/go/log"
	"github.com/stretchr/testify/require"
)

type fakeRedis struct {
	output string
	err    error
}

func (f fakeRedis) Set(context.Context, string, interface{}, time.Duration) error { return nil }
func (f fakeRedis) Get(context.Context, string) (string, error)                   { return f.output, f.err }
func (f fakeRedis) Delete(context.Context, ...string) error                       { return nil }
func (f fakeRedis) Exists(context.Context, string) (bool, error)                  { return false, nil }
func (f fakeRedis) TTL(context.Context, string) (time.Duration, error)            { return 0, nil }
func (f fakeRedis) Close() error                                                  { return nil }
func (f fakeRedis) Ping(context.Context) error                                    { return nil }
func (f fakeRedis) GetClient() *goredis.Client                                    { return nil }

func newTestRepo(r fakeRedis) *redisRepo {
	l := log.NewZapLogger(log.ZapConfig{Level: log.LevelFatal, Mode: log.ModeProduction, Encoding: log.EncodingJSON})
	return NewRepository(r, l).(*redisRepo)
}

func TestNewRepository(t *testing.T) {
	tcs := map[string]struct {
		input  fakeRedis
		mock   struct{}
		output bool
		err    error
	}{
		"success": {output: true},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			output := NewRepository(tc.input, nil)

			require.Equal(t, tc.output, output != nil)
		})
	}
}

func TestListActive(t *testing.T) {
	ctx := context.Background()

	tcs := map[string]struct {
		input  fakeRedis
		mock   struct{}
		output []Domain
		err    error
	}{
		"success": {
			input:  fakeRedis{output: `[{"domain_code":"ev","display_name":"EV"}]`},
			output: []Domain{{DomainCode: "ev", DisplayName: "EV"}},
		},
		"redis_error": {
			input: fakeRedis{err: errors.New("redis error")},
			err:   errors.New("domain.ListActive: redis GET smap:domains: redis error"),
		},
		"empty": {
			input: fakeRedis{},
			err:   errors.New("domain.ListActive: key smap:domains not found in Redis (analysis-srv may not have started)"),
		},
		"bad_json": {
			input: fakeRedis{output: `{`},
			err:   errors.New("domain.ListActive: unmarshal: unexpected end of JSON input"),
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			repo := newTestRepo(tc.input)

			output, err := repo.ListActive(ctx)

			if tc.err != nil {
				require.EqualError(t, err, tc.err.Error())
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.output, output)
		})
	}
}

func TestExists(t *testing.T) {
	ctx := context.Background()

	tcs := map[string]struct {
		input  string
		mock   fakeRedis
		output bool
		err    error
	}{
		"exists": {
			input:  "ev",
			mock:   fakeRedis{output: `[{"domain_code":"ev","display_name":"EV"}]`},
			output: true,
		},
		"not_exists": {
			input: "retail",
			mock:  fakeRedis{output: `[{"domain_code":"ev","display_name":"EV"}]`},
		},
		"list_error": {
			input: "ev",
			mock:  fakeRedis{err: errors.New("redis error")},
			err:   errors.New("domain.ListActive: redis GET smap:domains: redis error"),
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			repo := newTestRepo(tc.mock)

			output, err := repo.Exists(ctx, tc.input)

			if tc.err != nil {
				require.EqualError(t, err, tc.err.Error())
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.output, output)
		})
	}
}
