package esutils

import (
	"context"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestEs_BulkDelete(t *testing.T) {
	es, terminator := setupSubTest()
	defer terminator()
	es.BulkDelete(context.Background(), []string{"9seTXHoBNx091WJ2QCh5", "9seTXHoBNx091WJ2QCh6"})
	count, _ := es.Count(context.Background(), nil)
	assert.EqualValues(t, 1, count)
}
