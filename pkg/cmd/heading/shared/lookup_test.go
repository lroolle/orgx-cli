package shared

import (
	"testing"
)

func TestParseRefFromArg(t *testing.T) {
	tests := []struct {
		name    string
		arg     string
		want    Ref
	}{
		{
			name: "ID ref",
			arg:  "notes.org::ID:abc123",
			want: Ref{Path: "notes.org", RefType: RefTypeID, Value: "abc123"},
		},
		{
			name: "Hash ref",
			arg:  "README.md::H:1234abcd",
			want: Ref{Path: "README.md", RefType: RefTypeHash, Value: "1234abcd"},
		},
		{
			name: "Outline ref",
			arg:  "notes.org::/Projects/CLI",
			want: Ref{Path: "notes.org", RefType: RefTypeOutline, Value: "Projects/CLI"},
		},
		{
			name: "Plain path",
			arg:  "notes.org",
			want: Ref{Path: "notes.org", RefType: RefTypePath},
		},
		{
			name: "Path with spaces",
			arg:  "/path/to/my notes.org",
			want: Ref{Path: "/path/to/my notes.org", RefType: RefTypePath},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseRefFromArg(tt.arg)
			if err != nil {
				t.Fatalf("ParseRefFromArg() error = %v", err)
			}

			if got.Path != tt.want.Path {
				t.Errorf("Path = %q, want %q", got.Path, tt.want.Path)
			}
			if got.RefType != tt.want.RefType {
				t.Errorf("RefType = %v, want %v", got.RefType, tt.want.RefType)
			}
			if got.Value != tt.want.Value {
				t.Errorf("Value = %q, want %q", got.Value, tt.want.Value)
			}
		})
	}
}

func TestRefString(t *testing.T) {
	tests := []struct {
		ref  Ref
		want string
	}{
		{
			ref:  Ref{Path: "notes.org", RefType: RefTypeID, Value: "abc123"},
			want: "notes.org::ID:abc123",
		},
		{
			ref:  Ref{Path: "README.md", RefType: RefTypeHash, Value: "1234"},
			want: "README.md::H:1234",
		},
		{
			ref:  Ref{Path: "notes.org", RefType: RefTypeOutline, Value: "Projects"},
			want: "notes.org::/Projects",
		},
		{
			ref:  Ref{Path: "notes.org", RefType: RefTypePath},
			want: "notes.org",
		},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.ref.String()
			if got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}
