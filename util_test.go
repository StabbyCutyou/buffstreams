package buffstreams

import "testing"

func TestMessageSizeToBitLength(t *testing.T) {
	cases := []struct {
		input, output int
	}{
		{8, 1},
		{32, 1},
		{64, 1},
		{255, 1},
		{256, 2},
		{257, 2},
		{512, 2},
		{2048, 2},
		{4096, 2},
		{8192, 2},
	}

	for _, c := range cases {
		length := messageSizeToBitLength(c.input)
		if length != c.output {
			t.Errorf("Bit Length incorrect. For message size %d, got %d, expected %d", c.input, length, c.output)
		}
	}
}
