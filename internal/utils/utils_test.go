package utils

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func isEven(n int) bool { return n%2 == 0 }

func TestFilterEmpty(t *testing.T) {
	assert.Empty(t, Filter(nil, isEven))
}

func TestFilterKeepsOrder(t *testing.T) {
	assert.Equal(t, []int{4, 2, 2}, Filter([]int{1, 4, 3, 2, 2}, isEven))
}

func TestMapEmpty(t *testing.T) {
	assert.Empty(t, Map(nil, strconv.Itoa))
}

func TestMapKeepsDuplicates(t *testing.T) {
	assert.Equal(t, []string{"1", "1", "2"}, Map([]int{1, 1, 2}, strconv.Itoa))
}

func repeat(n int) []int {
	out := make([]int, n)
	for i := range out {
		out[i] = n
	}
	return out
}

func TestFlatMapEmpty(t *testing.T) {
	assert.Empty(t, FlatMap(nil, repeat))
}

func TestFlatMapConcatenates(t *testing.T) {
	assert.Equal(t, []int{1, 2, 2, 3, 3, 3}, FlatMap([]int{1, 0, 2, 3}, repeat))
}

func TestUniqueEmpty(t *testing.T) {
	assert.Empty(t, Unique([]string(nil)))
}

func TestUniqueKeepsFirstOccurrence(t *testing.T) {
	assert.Equal(t, []string{"b", "a", "c"}, Unique([]string{"b", "a", "b", "c", "a"}))
}

type pair struct {
	key   string
	value int
}

func pairKey(p pair) string { return p.key }

func TestUniqueByEmpty(t *testing.T) {
	assert.Empty(t, UniqueBy(nil, pairKey))
}

func TestUniqueByKeepsFirstForKey(t *testing.T) {
	got := UniqueBy([]pair{{"a", 1}, {"b", 2}, {"a", 3}}, pairKey)
	assert.Equal(t, []pair{{"a", 1}, {"b", 2}}, got)
}

func TestGroupByEmpty(t *testing.T) {
	assert.Empty(t, GroupBy(nil, pairKey))
}

func TestGroupByKeepsOrderWithinGroup(t *testing.T) {
	got := GroupBy([]pair{{"a", 1}, {"b", 2}, {"a", 3}}, pairKey)
	assert.Equal(t, map[string][]pair{"a": {{"a", 1}, {"a", 3}}, "b": {{"b", 2}}}, got)
}

func TestToSetEmpty(t *testing.T) {
	assert.Empty(t, ToSet([]string(nil)))
}

func TestToSetCollapsesDuplicates(t *testing.T) {
	assert.Equal(t, map[string]bool{"a": true, "b": true}, ToSet([]string{"a", "b", "a"}))
}

func TestSortedKeysEmpty(t *testing.T) {
	assert.Empty(t, SortedKeys(map[string]int(nil)))
}

func TestSortedKeys(t *testing.T) {
	assert.Equal(t, []string{"a", "b", "c"}, SortedKeys(map[string]int{"c": 1, "a": 2, "b": 3}))
}

func TestSortedUniqueEmpty(t *testing.T) {
	assert.Empty(t, SortedUnique([]string(nil)))
}

func TestSortedUniqueDropsDuplicates(t *testing.T) {
	assert.Equal(t, []string{"a", "b", "c"}, SortedUnique([]string{"c", "a", "b", "a", "c"}))
}
