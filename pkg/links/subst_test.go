package links

import (
	"testing"

	pb "jdtw.dev/links/proto/links"
)

func TestSubst(t *testing.T) {
	tests := []struct {
		s          string
		subs       []string
		want       string
		wantUnused []string
		wantErr    bool
	}{
		{
			s:          "",
			subs:       nil,
			want:       "",
			wantUnused: nil,
		},
		{
			s:          "foo",
			subs:       []string{"bar"},
			want:       "foo",
			wantUnused: []string{"bar"},
		},
		{
			s:          "{0}",
			subs:       []string{"foo"},
			want:       "foo",
			wantUnused: []string{},
		},
		{
			s:          "foo/{0}",
			subs:       []string{"bar", "baz"},
			want:       "foo/bar",
			wantUnused: []string{"baz"},
		},
		{
			s:          "{1}{0}",
			subs:       []string{"bar", "foo"},
			want:       "foobar",
			wantUnused: []string{},
		},
		{
			s:          "{1}",
			subs:       []string{"foo", "bar"},
			want:       "bar",
			wantUnused: []string{"foo"},
		},
		{
			s:       "{0}",
			subs:    []string{},
			wantErr: true,
		},
	}

Tests:
	for _, tc := range tests {
		le := &pb.LinkEntry{
			Link: &pb.Link{Uri: tc.s},
		}
		le.RequiredPaths = requiredPaths(le.Link)

		got, gotUnused, err := subst(le, tc.subs)
		if tc.wantErr && err == nil {
			t.Errorf("subst(%v, %v) = %v, %v, want error", tc.s, tc.subs, got, gotUnused)
			continue
		}
		if err != nil {
			if !tc.wantErr {
				t.Errorf("subst(%v, %v) failed: %v", tc.s, tc.subs, err)
			}
			continue
		}
		if got != tc.want {
			t.Errorf("subst(%v, %v) = %v, _; want %v, _", tc.s, tc.subs, got, tc.want)
			continue
		}
		if g, w := len(gotUnused), len(tc.wantUnused); g != w {
			t.Errorf("subst(%v, %v) = _, %v; want _, %v", tc.s, tc.subs, gotUnused, tc.wantUnused)
			continue
		}
		for i, got := range gotUnused {
			if got != tc.wantUnused[i] {
				t.Errorf("subst(%v, %v) = _, %v; want _, %v", tc.s, tc.subs, gotUnused, tc.wantUnused)
				continue Tests
			}
		}
	}
}

func TestIdx(t *testing.T) {
	tests := []struct {
		in   string
		want int
	}{
		{"{0}", 0},
		{"{1}", 1},
		{"{10}", 10},
		{"{12}", 12},
		{"{123}", 123},
		{"{999}", 999},
		{"{1000}", 1000},
	}

	for _, tc := range tests {
		if got := idx([]byte(tc.in)); got != tc.want {
			t.Errorf("matchInt(%s) = %d, want %d", tc.in, got, tc.want)
		}
	}
}
