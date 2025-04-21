package prefetch_test

import (
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/locker"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/prefetch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQueue_Push(t *testing.T) {
	locker.EnableInvariantsCheck()
	q := prefetch.NewQueue()

	q.Push(4)
	q.Push(5)

	assert.Equal(t, 4, q.Peek())
	assert.False(t, q.IsEmpty())
}

func TestQueue_Pop(t *testing.T) {
	locker.EnableInvariantsCheck()
	q := prefetch.NewQueue()

	q.Push(4)
	q.Push(5)
	require.Equal(t, 4, q.Peek()) // Use require here as the rest depends on this
	require.False(t, q.IsEmpty())

	val := q.Pop()
	assert.Equal(t, 4, val)
	assert.Equal(t, 5, q.Peek())

	val = q.Pop()
	assert.Equal(t, 5, val)
	assert.Nil(t, q.Peek())
	assert.True(t, q.IsEmpty())

	val = q.Pop()
	assert.Nil(t, val)
	assert.True(t, q.IsEmpty()) // Also check IsEmpty after popping from empty
}

func TestQueue_Peek(t *testing.T) {
	locker.EnableInvariantsCheck()
	q := prefetch.NewQueue()

	val := q.Peek()
	assert.Nil(t, val)

	q.Push(4)

	val = q.Peek()
	assert.Equal(t, 4, val)

	// Peek again to ensure it doesn't remove the element
	val = q.Peek()
	assert.Equal(t, 4, val)
	assert.False(t, q.IsEmpty())
	assert.Equal(t, 1, q.Len())
}

func TestQueue_IsEmpty(t *testing.T) {
	locker.EnableInvariantsCheck()
	q := prefetch.NewQueue()

	assert.True(t, q.IsEmpty())

	q.Push(4)
	assert.False(t, q.IsEmpty())

	val := q.Pop()
	assert.Equal(t, 4, val) // Verify the popped value
	assert.True(t, q.IsEmpty())

	// Check again after popping from empty
	val = q.Pop()
	assert.Nil(t, val)
	assert.True(t, q.IsEmpty())
}

func TestQueue_Len(t *testing.T) {
	locker.EnableInvariantsCheck()
	q := prefetch.NewQueue()

	assert.Equal(t, 0, q.Len())

	q.Push(4)
	assert.Equal(t, 1, q.Len())

	q.Push(5)
	assert.Equal(t, 2, q.Len())

	q.Push(6)
	assert.Equal(t, 3, q.Len())

	val := q.Pop()
	assert.Equal(t, 4, val) // Verify popped value
	assert.Equal(t, 2, q.Len())

	val = q.Pop()
	assert.Equal(t, 5, val) // Verify popped value
	assert.Equal(t, 1, q.Len())

	val = q.Pop()
	assert.Equal(t, 6, val) // Verify popped value
	assert.Equal(t, 0, q.Len())

	// Check Len after popping from empty
	val = q.Pop()
	assert.Nil(t, val)
	assert.Equal(t, 0, q.Len())
}
