package hush

import "testing"

func TestPathDescendant(t *testing.T) {
	tests := map[string][]string{
		// parent -> descendants
		"a":   {"a/b", "a/c", "a/b/c/d"},
		"a/b": {"a/b/foo", "a/b/foo/bar/baz"},
	}
	for parent, descendants := range tests {
		p := NewPath(parent)
		for _, descendant := range descendants {
			d := NewPath(descendant)
			if !p.HasDescendant(d) {
				t.Errorf("%q should be descended from %q", d, p)
			}
		}
	}

	a := NewPath("a")
	ab := NewPath("ab")
	if a.HasDescendant(a) {
		t.Errorf("a should not be a descendant of itself")
	}
	if a.HasDescendant(ab) {
		t.Errorf("ab should not be a descendant of a")
	}
	if ab.HasDescendant(a) {
		t.Errorf("a should not be a descendant of ab")
	}
}
