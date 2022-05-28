package links

import (
	"fmt"
	"regexp"

	pb "jdtw.dev/links/proto/links"
)

var replacement = regexp.MustCompile(`{\d+}`)

// requiredPaths pre-processes a link before it gets added to the database,
// calculating the number of required path segments to satisfy the
// substitution. For example, "http://example.com/{1}/{0}" requires two paths,
// and "http://example.com/{99}" requires 100 paths.
func requiredPaths(l *pb.Link) int32 {
	n := -1
	for _, found := range replacement.FindAll([]byte(l.Uri), -1) {
		if m := idx(found); m > n {
			n = m
		}
	}
	return int32(n + 1)
}

// subst takes a link entry and the paths from the incoming request, and
// performs {\d+} substitutions, returning the link URI after substitution
// and any remaining unused path segments. For example,
// subst("http://example.com/{1}/{0}", ["foo", "bar", "baz"]) returns
// "http://example.com/bar/foo", ["baz"].
func subst(le *pb.LinkEntry, paths []string) (string, []string, error) {
	if le.RequiredPaths == 0 {
		return le.Link.Uri, paths, nil
	}

	if len(paths) < int(le.RequiredPaths) {
		return "", nil, fmt.Errorf("got %d params, want %d", len(paths), le.RequiredPaths)
	}

	used := make([]bool, len(paths))
	replaced := replacement.ReplaceAllFunc([]byte(le.Link.Uri), func(m []byte) []byte {
		i := idx(m)
		used[i] = true
		return []byte(paths[i])
	})

	unused := make([]string, 0, len(paths))
	for i, p := range paths {
		if !used[i] {
			unused = append(unused, p)
		}
	}
	return string(replaced), unused, nil
}

// idx takes a replaceemnt index of the form `{\d+}` and returns
// the integer inside the braces.
func idx(m []byte) int {
	n := 0
	for i := 1; i < len(m)-1; i++ {
		// No need to check for non-digit characters, since `m` matches
		// `\d+`.
		n = n*10 + int(m[i]-'0')
	}
	return n
}
