package yahoo

import "fmt"

// PageOptions controls pagination for Yahoo collection endpoints. Start is the
// zero-based offset; Count is the page size. A Count <= 0 requests Yahoo's
// default page size. The zero value (Start 0, Count 0) means "no explicit
// pagination" and yields Yahoo's default response.
type PageOptions struct {
	Start int
	Count int
}

// suffix renders the Yahoo ;start=N;count=M path segment, or "" for the zero
// value (letting Yahoo apply its default).
func (p PageOptions) suffix() string {
	if p.Start <= 0 && p.Count <= 0 {
		return ""
	}
	s := fmt.Sprintf(";start=%d", max(0, p.Start))
	if p.Count > 0 {
		s += fmt.Sprintf(";count=%d", p.Count)
	}
	return s
}
