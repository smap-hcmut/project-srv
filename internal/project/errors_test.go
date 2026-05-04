package project

import (
	"errors"
	"testing"

	"project-srv/pkg/microservice"

	"github.com/stretchr/testify/require"
)

func TestMapLifecycleClientError(t *testing.T) {
	tcs := map[string]struct {
		input  error
		mock   struct{}
		output error
		err    error
	}{
		"bad_request": {
			input:  microservice.ErrBadRequest,
			output: ErrLifecycleManagerRejected,
		},
		"unauthorized": {
			input:  microservice.ErrUnauthorized,
			output: ErrLifecycleManagerUnauthorized,
		},
		"forbidden": {
			input:  microservice.ErrForbidden,
			output: ErrLifecycleManagerForbidden,
		},
		"default": {
			input:  errors.New("network error"),
			output: ErrLifecycleManagerFailed,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			output := MapLifecycleClientError(tc.input)

			require.Equal(t, tc.output, output)
		})
	}
}
