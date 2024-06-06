package models

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUser_AddMeeting(t *testing.T) {
	type args struct {
		intervals []Meeting
		t         Meeting
	}

	type testcase struct {
		name string
		args args

		wantIdx int
		wantOk  bool
	}

	tests := [...]testcase{
		{
			name: "add to empty",
			args: args{
				intervals: nil,
				t:         [2]int64{2, 4},
			},
			wantIdx: 0,
			wantOk:  true,
		},
		{
			name: "add to the end",
			args: args{
				intervals: []Meeting{{0, 2}, {2, 3}},
				t:         [2]int64{3, 4},
			},
			wantIdx: 2,
			wantOk:  true,
		},
		{
			name: "add to the middle",
			args: args{
				intervals: []Meeting{{0, 2}, {2, 3}, {4, 5}},
				t:         [2]int64{3, 4},
			},
			wantIdx: 2,
			wantOk:  true,
		},
		{
			name: "add to the beginning",
			args: args{
				intervals: []Meeting{{2, 3}, {3, 4}},
				t:         [2]int64{0, 1},
			},
			wantIdx: 0,
			wantOk:  true,
		},
		{
			name: "overlap first",
			args: args{
				intervals: []Meeting{{2, 3}, {3, 4}},
				t:         [2]int64{0, 3},
			},
			wantIdx: 0,
			wantOk:  false,
		},

		{
			name: "overlap last",
			args: args{
				intervals: []Meeting{{2, 3}, {3, 5}},
				t:         [2]int64{4, 6},
			},
			wantIdx: 2,
			wantOk:  false,
		},
		{
			name: "no space intersect one",
			args: args{
				intervals: []Meeting{{0, 2}, {2, 3}},
				t:         [2]int64{1, 2},
			},
			wantIdx: 1,
			wantOk:  false,
		},
		{
			name: "no space intersect two",
			args: args{
				intervals: []Meeting{{0, 2}, {2, 3}},
				t:         [2]int64{1, 2},
			},
			wantIdx: 1,
			wantOk:  false,
		},
		{
			name: "no space intersect many",
			args: args{
				intervals: []Meeting{{0, 1}, {1, 2}, {2, 3}, {4, 6}},
				t:         [2]int64{1, 8},
			},
			wantIdx: 1,
			wantOk:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := User{Assigned: tt.args.intervals}
			gotIdx, gotOk := u.AddMeeting(tt.args.t)
			require.Equal(t, tt.wantIdx, gotIdx)
			require.Equal(t, tt.wantOk, gotOk)

			require.NotPanics(t, func() {
				_ = slices.Insert(u.Assigned, gotIdx, tt.args.t)
			})
		})
	}
}

func TestUser_FindAndDeleteMeeting(t *testing.T) {
	type testcase struct {
		name     string
		assigned []Meeting
		arg      Meeting
		want     []Meeting
		wantOk   bool
	}

	tests := [...]testcase{
		{
			name:   "no assigned",
			arg:    Meeting{1, 2},
			wantOk: false,
		},
		{
			name:     "has meet",
			assigned: []Meeting{{1, 2}, {3, 5}, {5, 6}},
			arg:      Meeting{3, 5},
			want:     []Meeting{{1, 2}, {5, 6}},
			wantOk:   true,
		},
		{
			name:     "mismatched start",
			assigned: []Meeting{{1, 2}, {3, 5}, {6, 7}},
			arg:      Meeting{4, 7},
			want:     []Meeting{{1, 2}, {3, 5}, {6, 7}},
			wantOk:   false,
		},
		{
			name:     "mismatched end",
			assigned: []Meeting{{1, 2}, {3, 5}, {5, 6}},
			arg:      Meeting{3, 4},
			want:     []Meeting{{1, 2}, {3, 5}, {5, 6}},
			wantOk:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := User{Assigned: tt.assigned}
			got, gotOk := u.FindAndDeleteMeeting(tt.arg)
			require.ElementsMatch(t, tt.want, got)
			require.Equal(t, tt.wantOk, gotOk)
		})
	}
}
