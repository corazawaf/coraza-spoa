package server

import (
	"errors"
	"testing"

	"github.com/corazawaf/coraza/v3/types"
	"github.com/stretchr/testify/assert"
)

func TestInterruptionError(t *testing.T) {
	t.Run("should implement error interface", func(t *testing.T) {
		err := &ErrInterrupted{
			Interruption: &types.Interruption{
				Data: "test",
			},
		}
		e := &ErrInterrupted{}
		assert.True(t, errors.As(err, &e))
		assert.Equal(t, e.Interruption.Data, "test")
	})
	t.Run("should not match interface", func(t *testing.T) {
		err := errors.New("test")
		e := &ErrInterrupted{}
		assert.False(t, errors.As(err, &e))
	})
}
