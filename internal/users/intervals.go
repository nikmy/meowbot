package users

import "sort"

func addInterval(intervals [][2]int64, t [2]int64) (int, bool) {
	n := len(intervals)

	if n == 0 {
		return 0, true
	}

	// find position for beginning to insert
	idx := sort.Search(n, func(i int) bool {
		return t[0] <= intervals[i][0]
	})

	if idx == n {
		// all intervals start earlier, check
		// overlap with last one's end
		return idx, t[0] >= intervals[n-1][1]
	}

	if t[1] > intervals[idx][0] {
		return idx, false
	}

	// check overlap with previous one
	if idx > 0 && t[0] < intervals[idx-1][1] {
		return idx, false
	}

	return idx, true
}
