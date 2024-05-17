package users

import "sort"

func addMeeting(scheduled []Meeting, toAdd Meeting) (int, bool) {
	n := len(scheduled)

	if n == 0 {
		return 0, true
	}

	// find position for beginning to insert
	idx := sort.Search(n, func(i int) bool {
		return toAdd[0] <= scheduled[i][0]
	})

	if idx == n {
		// all meetings start earlier, check
		// overlap with last one's end
		return idx, toAdd[0] >= scheduled[n-1][1]
	}

	if toAdd[1] > scheduled[idx][0] {
		return idx, false
	}

	// check overlap with previous one
	if idx > 0 && toAdd[0] < scheduled[idx-1][1] {
		return idx, false
	}

	return idx, true
}
