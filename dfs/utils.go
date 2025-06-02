// Package dfs provides common helper functions used across DFS, cycle detection, and topological sort implementations.
// These utilities offer string-slice operations and Booth's minimal-rotation algorithm.
package dfs

import (
	"strings"
)

// IndexOf returns the first index of val in s, or -1 if not found.
// Time Complexity: O(n) where n = len(s).
func IndexOf(s []string, val string) int {
	for i, x := range s { // iterate through slice
		if x == val { // compare each element
			return i // return index when found
		}
	}

	return -1 // not found
}

// Reverse returns a new slice containing the elements of s in reverse order.
// Time Complexity: O(n).
func Reverse(s []string) []string {
	out := make([]string, len(s)) // allocate new slice of same length
	for i := range s {            // iterate indices
		out[i] = s[len(s)-1-i] // assign from opposite end
	}

	return out
}

// Compare lexicographically compares two equal-length string slices a and b.
// Returns -1 if a < b, 0 if equal, +1 if a > b.
// Comparison proceeds element-by-element like lexicographical order.
// Time Complexity: O(n).
func Compare(a, b []string) int {
	for i := range a { // both slices assumed same length
		if a[i] < b[i] {
			return -1 // first differing element a[i] < b[i]
		} else if a[i] > b[i] {
			return 1 // first differing element a[i] > b[i]
		}
	}

	return 0 // all elements equal
}

// JoinSig concatenates the elements of c with commas, producing a single string signature.
// Time Complexity: O(n + total length of elements).
func JoinSig(c []string) string {
	return strings.Join(c, ",") // built-in join
}

// MinimalRotation implements Booth's algorithm to find the lexicographically minimal rotation of s.
// It returns a new slice of length len(s) representing the minimal rotation in O(n) time.
// Algorithm overview:
// 1. Duplicate the sequence (doubled) to length 2n.
// 2. Maintain an array f of failure links initialized to -1.
// 3. Track candidate k = 0; for j from 1 to 2n-1, adjust k based on comparisons.
// 4. After scanning, extract the rotation starting at index k.
// Time Complexity: O(n).
func MinimalRotation(s []string) []string {
	doubled := append(s, s...) // duplicate sequence
	n := len(s)                // original length
	f := make([]int, 2*n)      // failure link array
	for i := range f {
		f[i] = -1 // initialize all to -1
	}
	k := 0                     // starting index of minimal rotation
	for j := 1; j < 2*n; j++ { // iterate through doubled sequence
		i := f[j-k-1] // failure link lookup
		for i != -1 && doubled[j] != doubled[k+i+1] {
			if doubled[j] < doubled[k+i+1] { // found smaller element
				k = j - i - 1 // update candidate k
			}
			i = f[i] // jump in failure links
		}
		if doubled[j] != doubled[k+i+1] { // mismatch or i == -1
			if doubled[j] < doubled[k] { // j-th element smaller than current candidate
				k = j // update k
			}
			f[j-k] = -1 // set failure at new position
		} else {
			f[j-k] = i + 1 // extend match length
		}
	}
	// extract minimal rotation of length n starting at k
	res := make([]string, n)
	for i := 0; i < n; i++ {
		res[i] = doubled[k+i] // copy each element
	}

	return res
}
