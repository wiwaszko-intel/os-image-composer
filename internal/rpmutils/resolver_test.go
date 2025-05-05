package rpmutils

import (
	"reflect"
	"sort"
	"testing"
)

// helper to compare two slices as sets
func equalUnordered(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	sa := append([]string(nil), a...)
	sb := append([]string(nil), b...)
	sort.Strings(sa)
	sort.Strings(sb)
	return reflect.DeepEqual(sa, sb)
}

func TestResolveDependencies_SimpleChain(t *testing.T) {
	// A → requires B → requires C
	idx := &Index{
		Provides: map[string][]string{
			"A": {"pkgA.rpm"},
			"B": {"pkgB.rpm"},
			"C": {"pkgC.rpm"},
		},
		Requires: map[string][]string{
			"pkgA.rpm": {"B"},
			"pkgB.rpm": {"C"},
			"pkgC.rpm": {},
		},
	}
	sel := []string{"pkgA.rpm"}
	want := []string{"pkgA.rpm", "pkgB.rpm", "pkgC.rpm"}
	got := ResolveDependencies(sel, idx)
	if !equalUnordered(got, want) {
		t.Errorf("ResolveDependencies(chain) = %v; want %v", got, want)
	}
}

func TestResolveDependencies_MultipleProviders(t *testing.T) {
	// A requires X, X provided by P1 and P2, P2 requires Y
	idx := &Index{
		Provides: map[string][]string{
			"A": {"pkgA.rpm"},
			"X": {"pkgP1.rpm", "pkgP2.rpm"},
			"Y": {"pkgY.rpm"},
		},
		Requires: map[string][]string{
			"pkgA.rpm":   {"X"},
			"pkgP1.rpm":  {},
			"pkgP2.rpm":  {"Y"},
			"pkgY.rpm":   {},
		},
	}
	sel := []string{"pkgA.rpm"}
	// either pkgP1 or pkgP2 could satisfy X; both should be included
	want := []string{"pkgA.rpm", "pkgP1.rpm", "pkgP2.rpm", "pkgY.rpm"}
	got := ResolveDependencies(sel, idx)
	if !equalUnordered(got, want) {
		t.Errorf("ResolveDependencies(multi-provider) = %v; want %v", got, want)
	}
}

func TestResolveDependencies_NoDeps(t *testing.T) {
	// no requires
	idx := &Index{
		Provides: map[string][]string{},
		Requires: map[string][]string{},
	}
	sel := []string{"pkgX.rpm"}
	// if pkgX not in index, it should still appear
	want := []string{"pkgX.rpm"}
	got := ResolveDependencies(sel, idx)
	if !equalUnordered(got, want) {
		t.Errorf("ResolveDependencies(no index) = %v; want %v", got, want)
	}
}
