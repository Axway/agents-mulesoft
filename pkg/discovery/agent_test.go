package discovery

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCleanTags(t *testing.T) {
	input := "discoverone,discovertwo,discoverthree "
	assert.Equal(t, []string{"discoverone", "discovertwo", "discoverthree"}, cleanTags(input))
}
