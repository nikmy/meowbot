package models

import (
	"slices"
	"sort"
)

func (u User) AddMeeting(meeting Meeting) (int, bool) {
	scheduled := u.Assigned

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

func (u User) FindAndDeleteMeeting(meeting Meeting) ([]Meeting, bool) {
	idx := sort.Search(len(u.Assigned), func(i int) bool {
		return u.Assigned[i][0] >= meeting[0]
	})
	if idx == len(u.Assigned) {
		return u.Assigned, false
	}

	found := u.Assigned[idx]

	if found[0] != meeting[0] || found[1] != meeting[1] {
		return u.Assigned, false
	}

	return slices.Delete(u.Assigned, idx, idx+1), true
}
