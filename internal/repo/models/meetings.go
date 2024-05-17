package models

import "sort"

func (u User) AddMeeting(meeting Meeting) (int, bool) {
	scheduled := u.Meetings

	n := len(scheduled)

	if n == 0 {
		return 0, true
	}

	// find position for beginning to insert
	idx := sort.Search(n, func(i int) bool {
		return meeting[0] <= scheduled[i][0]
	})

	if idx == n {
		// all meetings start earlier, check
		// overlap with last one's end
		return idx, meeting[0] >= scheduled[n-1][1]
	}

	if meeting[1] > scheduled[idx][0] {
		return idx, false
	}

	// check overlap with previous one
	if idx > 0 && meeting[0] < scheduled[idx-1][1] {
		return idx, false
	}

	return idx, true
}
