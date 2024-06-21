package settings

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFieldOptions(t *testing.T) {
	type input struct {
		A string
		B struct {
			C string
			D string
		}
	}
	tests := []struct {
		name  string
		input any
		want  []string
		want1 map[string]field
	}{
		{
			name:  "ok",
			input: input{},
			want:  []string{"A", "B.C", "B.D"},
		},
		{
			name:  "ok",
			input: &input{},
			want:  []string{"A", "B.C", "B.D"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FieldOptions(tt.input)
			keys := make([]string, 0, len(got))
			for k := range got {
				keys = append(keys, k)
			}

			assert.Equal(t, tt.want, keys)
		})
	}
}
