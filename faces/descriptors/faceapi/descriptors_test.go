package faceapi

import (
	"fmt"
	"testing"

	"github.com/gildasch/gildas-ai/imageutils"
	"github.com/stretchr/testify/require"
)

func TestDescriptors(t *testing.T) {
	d, err := NewDescriptor()
	require.NoError(t, err)

	dd := map[string]*Descriptors{}
	for _, i := range []string{"1", "2", "5", "8", "10", "11", "4-face-1-cropped"} {
		img, err := imageutils.FromFile(fmt.Sprintf("%s.jpg", i))
		require.NoError(t, err)
		dd[i], err = d.Compute(img)
		require.NoError(t, err)
	}

	for a, da := range dd {
		for b, db := range dd {
			fmt.Printf("%s vs %s: %f\n", a, b, da.DistanceTo(db))
		}
	}
}