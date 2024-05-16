package telegram

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/nikmy/meowbot/internal/interviews"
)

func Test_getNeededNotifications(t *testing.T) {
	type fields struct {
		notifyBefore []int64
		notifyPeriod int64
	}

	type args struct {
		now      int64
		upcoming []interviews.Interview
	}

	type testcase struct {
		name   string
		fields fields
		args   args
		want   []notification
	}

	both := []interviews.Role{interviews.RoleInterviewer, interviews.RoleCandidate}

	tests := [...]testcase{
		{
			name: "no upcoming",
			fields: fields{
				notifyBefore: []int64{10, 100, 1},
				notifyPeriod: 5,
			},
			args: args{
				now:      1000,
				upcoming: []interviews.Interview{},
			},
			want: []notification{},
		},
		{
			name: "need notify first time",
			fields: fields{
				notifyBefore: []int64{10, 100, 1},
				notifyPeriod: 5,
			},
			args: args{
				now: 1000,
				upcoming: []interviews.Interview{
					{Interval: [2]int64{1020, 1100}},
				},
			},
			want: []notification{
				{
					Interview:  interviews.Interview{Interval: [2]int64{1020, 1100}},
					Recipients: both,
					NotifyTime: 920,
					LeftTime:   100 * time.Millisecond,
				},
			},
		},
		{
			name: "too early to notify",
			fields: fields{
				notifyBefore: []int64{10, 100, 1},
				notifyPeriod: 5,
			},
			args: args{
				now: 1000,
				upcoming: []interviews.Interview{
					{Interval: [2]int64{1110, 1200}},
				},
			},
			want: []notification{},
		},
		{
			name: "already notified",
			fields: fields{
				notifyBefore: []int64{10, 300, 100},
				notifyPeriod: 5,
			},
			args: args{
				now: 1000,
				upcoming: []interviews.Interview{
					{
						Interval: [2]int64{1050, 1100},
						LastNotification: &interviews.NotificationLog{
							UnixTime: 950,
							Notified: [2]bool{true, true},
						},
					},
				},
			},
			want: []notification{},
		},
		{
			name: "already notified at time",
			fields: fields{
				notifyBefore: []int64{10, 300, 100},
				notifyPeriod: 5,
			},
			args: args{
				now: 1000,
				upcoming: []interviews.Interview{
					{
						Interval: [2]int64{1002, 1010},
						LastNotification: &interviews.NotificationLog{
							UnixTime: 1002,
							Notified: [2]bool{true, true},
						},
					},
				},
			},
			want: []notification{},
		},
		{
			name: "notify at time",
			fields: fields{
				notifyBefore: []int64{10, 100, 1},
				notifyPeriod: 5,
			},
			args: args{
				now: 1000,
				upcoming: []interviews.Interview{
					{
						Interval: [2]int64{1004, 1100},
						LastNotification: &interviews.NotificationLog{
							UnixTime: 900,
							Notified: [2]bool{true},
						},
					},
				},
			},
			want: []notification{
				{
					Interview: interviews.Interview{
						Interval: [2]int64{1004, 1100},
						LastNotification: &interviews.NotificationLog{
							UnixTime: 900,
							Notified: [2]bool{true},
						},
					},
					Recipients: both,
					NotifyTime: 1004,
					LeftTime:   0,
				},
			},
		},
		{
			name: "notify because last sent earlier",
			fields: fields{
				notifyBefore: []int64{10, 100, 1},
				notifyPeriod: 5,
			},
			args: args{
				now: 1000,
				upcoming: []interviews.Interview{
					{
						Interval: [2]int64{1100, 1200},
						LastNotification: &interviews.NotificationLog{
							UnixTime: 900,
							Notified: [2]bool{true},
						},
					},
				},
			},
			want: []notification{
				{
					Interview: interviews.Interview{
						Interval: [2]int64{1100, 1200},
						LastNotification: &interviews.NotificationLog{
							UnixTime: 900,
							Notified: [2]bool{true},
						},
					},
					Recipients: both,
					NotifyTime: 1000,
					LeftTime:   100 * time.Millisecond,
				},
			},
		},
		{
			name: "notify only candidate",
			fields: fields{
				notifyBefore: []int64{10, 100, 1},
				notifyPeriod: 5,
			},
			args: args{
				now: 1000,
				upcoming: []interviews.Interview{
					{
						Interval: [2]int64{1050, 1100},
						LastNotification: &interviews.NotificationLog{
							UnixTime: 950,
							Notified: [2]bool{true},
						},
					},
				},
			},
			want: []notification{
				{
					Interview: interviews.Interview{
						Interval: [2]int64{1050, 1100},
						LastNotification: &interviews.NotificationLog{
							UnixTime: 950,
							Notified: [2]bool{true},
						},
					},
					Recipients: []interviews.Role{interviews.RoleCandidate},
					NotifyTime: 950,
					LeftTime:   100 * time.Millisecond,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			logMock := NewMockloggerImpl(ctrl)
			logMock.EXPECT().
				Warnf(gomock.Any()).AnyTimes()

			iMock, uMock := NewMockinterviewsApi(ctrl), NewMockusersApi(ctrl)

			period := time.Duration(tt.fields.notifyPeriod) * time.Millisecond
			before := make([]time.Duration, 0, len(tt.fields.notifyBefore))
			for _, i := range tt.fields.notifyBefore {
				before = append(before, time.Duration(i)*time.Millisecond)
			}

			cfg := Config{
				NotifyPeriod: period,
				NotifyBefore: before,
			}

			b := &Bot{log: logMock, interviews: iMock, users: uMock}
			b.applyNotifications(cfg)

			got := b.getNeededNotifications(tt.args.now, tt.args.upcoming)
			require.ElementsMatch(t, tt.want, got)
		})
	}
}
