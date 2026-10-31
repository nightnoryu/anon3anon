package handler

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAllowListEmptyAllowsEveryone(t *testing.T) {
	t.Parallel()

	a := NewAllowList(nil)
	assert.True(t, a.Allowed(1))
	assert.True(t, a.Allowed(123456))
}

func TestAllowListRestrictsToListedIDs(t *testing.T) {
	t.Parallel()

	a := NewAllowList([]int64{10, 20})
	assert.True(t, a.Allowed(10))
	assert.True(t, a.Allowed(20))
	assert.False(t, a.Allowed(30))
}
