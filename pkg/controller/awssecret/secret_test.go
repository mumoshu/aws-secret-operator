package awssecret

import (
	"github.com/google/go-cmp/cmp"
	"testing"
)

func TestAWSSecretValueToMap(t *testing.T) {
	type testcase struct {
		input string
		want  map[string]string
	}

	testcases := []testcase{
		{
			input: "foo",
			want:  map[string]string{"data": "foo"},
		},
		{
			input: `{"bar": "BAR"}`,
			want:  map[string]string{"bar": "BAR"},
		},
		{
			input: `{"host":"abcdefg","port":123}`,
			want:  map[string]string{"host": "abcdefg", "port": "123"},
		},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.input, func(t *testing.T) {
			got, err := awsSecretValueToMap(tc.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("unexpected result:\n%s", diff)
			}
		})
	}
}
