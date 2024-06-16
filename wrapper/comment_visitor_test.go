package wrapper

import (
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommentVisitor(t *testing.T) {
	expectedTargets := []string{
		"MagicNamedParamsAndResults",
		"MagicUnnamedAndNamedParamsAndResults",
		"MagicUnderscoreNames",
		"MagicNoParams",
		"MagicNoResult",
	}


    wd := copyFilesHelper(t)
    cv, err := NewCommentVisitor(path.Join(wd, "magic_comment.go"))
    require.NoError(t,err)

    err = cv.Walk()
    require.NoError(t, err)
    
    assert.ElementsMatch(t, expectedTargets, cv.Targets())
}
