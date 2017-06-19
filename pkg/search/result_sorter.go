package search

import (
	"sort"

	"k8s.io/helm/cmd/helm/search"
)

type sorter struct {
	list []*search.Result
	less func(int, int) bool
}

func (s *sorter) Len() int           { return len(s.list) }
func (s *sorter) Less(i, j int) bool { return s.less(i, j) }
func (s *sorter) Swap(i, j int)      { s.list[i], s.list[j] = s.list[j], s.list[i] }

// SortByRevision returns the list of releases sorted by a
// release's revision number (release.Version).
func SortByRevision(list []*search.Result) {
	s := &sorter{list: list}
	s.less = func(i, j int) bool {
		vi := s.list[i].Chart.Version
		vj := s.list[j].Chart.Version
		return vi > vj
	}
	sort.Sort(s)
}

func SortByName(list []*search.Result) {
	s := &sorter{list: list}
	s.less = func(i, j int) bool {
		vi := s.list[i].Chart.Name
		vj := s.list[j].Chart.Name
		return vi < vj
	}
	sort.Sort(s)
}
