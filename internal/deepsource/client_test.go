package deepsource

import "testing"

func TestMatchPath(t *testing.T) {
	cases := []struct {
		pattern string
		path    string
		want    bool
	}{
		{"internal/**", "internal/foo.go", true},
		{"internal/**", "internal/foo/bar.go", true},
		{"internal/**", "cmd/foo.go", false},
		{"**/*_test.go", "internal/foo_test.go", true},
		{"**/*_test.go", "deeply/nested/dir/x_test.go", true},
		{"**/*_test.go", "internal/foo.go", false},
		{"vendor/**", "vendor/x/y/z.go", true},
		{"vendor/**", "internal/vendor/x.go", false},
		{"*.go", "foo.go", true},
		{"*.go", "sub/foo.go", false},
		{"a/b/c.go", "a/b/c.go", true},
		{"a/b/c.go", "a/b/d.go", false},
		{"**", "anything/at/all.go", true},
	}
	for _, c := range cases {
		if got := matchPath(c.pattern, c.path); got != c.want {
			t.Errorf("matchPath(%q, %q) = %v, want %v", c.pattern, c.path, got, c.want)
		}
	}
}

func TestPathAllowed(t *testing.T) {
	t.Run("no filters allows everything", func(t *testing.T) {
		if !pathAllowed("any/path.go", nil, nil) {
			t.Error("expected true with no filters")
		}
	})
	t.Run("include must match", func(t *testing.T) {
		if !pathAllowed("internal/x.go", []string{"internal/**"}, nil) {
			t.Error("expected true: path matches include")
		}
		if pathAllowed("cmd/x.go", []string{"internal/**"}, nil) {
			t.Error("expected false: path does not match include")
		}
	})
	t.Run("exclude wins over include", func(t *testing.T) {
		if pathAllowed("internal/x_test.go",
			[]string{"internal/**"},
			[]string{"**/*_test.go"}) {
			t.Error("expected false: matches exclude")
		}
	})
}
