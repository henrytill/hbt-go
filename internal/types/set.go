package types

import (
	"maps"
	"slices"
)

// Set is the map[T]struct{} idiom under a name. Its constraint is what a map
// key requires and no more; operations that need an ordering take one.
type Set[T comparable] map[T]struct{}

func NewSet[T comparable](items ...T) Set[T] {
	s := make(Set[T], len(items))
	for _, item := range items {
		s[item] = struct{}{}
	}
	return s
}

// Merge adds the elements of other to s and returns the result. Like append,
// it may modify s in place, so callers must use the returned value.
func (s Set[T]) Merge(other Set[T]) Set[T] {
	if len(other) == 0 {
		return s
	}
	if s == nil {
		s = make(Set[T], len(other))
	}
	maps.Copy(s, other)
	return s
}

func SortedSlice[T ~string](s Set[T]) []string {
	strs := make([]string, 0, len(s))
	for k := range s {
		strs = append(strs, string(k))
	}
	slices.Sort(strs)
	return strs
}

func sliceToSet[T ~string](slice []string) Set[T] {
	s := make(Set[T], len(slice))
	for _, str := range slice {
		if str != "" {
			s[T(str)] = struct{}{}
		}
	}
	return s
}
