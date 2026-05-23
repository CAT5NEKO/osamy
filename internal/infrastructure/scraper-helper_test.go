package infrastructure

import "testing"

func TestEnsureAbsoluteUrl(t *testing.T) {
	cases := []struct {
		input   string
		origin  string
		expect  string
	}{
		{input: "https://example.com/a.jpg", origin: "https://origin.example", expect: "https://example.com/a.jpg"},
		{input: "//cdn.example.com/a.jpg", origin: "https://origin.example", expect: "https://cdn.example.com/a.jpg"},
		{input: "/images/a.jpg", origin: "https://origin.example", expect: "https://origin.example/images/a.jpg"},
	}

	for _, tc := range cases {
		got := EnsureAbsoluteUrl(tc.input, tc.origin)
		if got != tc.expect {
			t.Fatalf("unexpected url: got %s want %s", got, tc.expect)
		}
	}
}
