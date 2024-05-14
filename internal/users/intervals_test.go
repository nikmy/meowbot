package users

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_addInterval(t *testing.T) {
	type args struct {
		intervals [][2]int64
		t         [2]int64
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
				intervals: [][2]int64{{0, 2}, {2, 3}},
				t:         [2]int64{3, 4},
			},
			wantIdx: 2,
			wantOk:  true,
		},
		{
			name: "add to the middle",
			args: args{
				intervals: [][2]int64{{0, 2}, {2, 3}, {4, 5}},
				t:         [2]int64{3, 4},
			},
			wantIdx: 2,
			wantOk:  true,
		},
		{
			name: "add to the beginning",
			args: args{
				intervals: [][2]int64{{2, 3}, {3, 4}},
				t:         [2]int64{0, 1},
			},
			wantIdx: 0,
			wantOk:  true,
		},
		{
			name: "overlap first",
			args: args{
				intervals: [][2]int64{{2, 3}, {3, 4}},
				t:         [2]int64{0, 3},
			},
			wantIdx: 0,
			wantOk:  false,
		},

		{
			name: "overlap last",
			args: args{
				intervals: [][2]int64{{2, 3}, {3, 5}},
				t:         [2]int64{4, 6},
			},
			wantIdx: 2,
			wantOk:  false,
		},
		{
			name: "no space intersect one",
			args: args{
				intervals: [][2]int64{{0, 2}, {2, 3}},
				t:         [2]int64{1, 2},
			},
			wantIdx: 1,
			wantOk:  false,
		},
		{
			name: "no space intersect two",
			args: args{
				intervals: [][2]int64{{0, 2}, {2, 3}},
				t:         [2]int64{1, 2},
			},
			wantIdx: 1,
			wantOk:  false,
		},
		{
			name: "no space intersect many",
			args: args{
				intervals: [][2]int64{{0, 1}, {1, 2}, {2, 3}, {4, 6}},
				t:         [2]int64{1, 8},
			},
			wantIdx: 1,
			wantOk:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotIdx, gotOk := addInterval(tt.args.intervals, tt.args.t)
			require.Equal(t, tt.wantIdx, gotIdx)
			require.Equal(t, tt.wantOk, gotOk)

			require.NotPanics(t, func() {
				tt.args.intervals = slices.Insert(tt.args.intervals, gotIdx, tt.args.t)
			})
		})
	}
}
