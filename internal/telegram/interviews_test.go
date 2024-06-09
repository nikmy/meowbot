package telegram

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vitaliy-ukiru/fsm-telebot"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/telebot.v3"

	"github.com/nikmy/meowbot/internal/repo/models"
)

func TestBot_showInterviews(t *testing.T) {
	type mocks struct {
		sender *telebot.User

		interviews []*models.Interview
		iErr       error

		c telebot.Context
		s fsm.Context
	}

	type want struct {
		fail bool
	}

	type testcase struct {
		name string
		mock mocks
		want want
	}

	tests := [...]testcase{
		{
			name: "no user",
			mock: mocks{},
			want: want{fail: true},
		},
		{
			name: "err while fetching interview",
			mock: mocks{
				sender: &telebot.User{ID: 42},
				iErr:   errors.New("mock"),
			},
			want: want{fail: true},
		},
		{
			name: "no assigned interviews",
			mock: mocks{
				sender: &telebot.User{ID: 42},
			},
			want: want{fail: false},
		},
		{
			name: "have assigned interviews",
			mock: mocks{
				sender: &telebot.User{ID: 42},
				interviews: []*models.Interview{
					{Meet: &[2]int64{10, 20}},
					{},
					{Meet: &[2]int64{20, 30}},
				},
			},
			want: want{fail: false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			cMock := NewMocktelebotContext(ctrl)

			cMock.EXPECT().Sender().Return(tt.mock.sender).Times(1)
			cMock.EXPECT().Send(gomock.Any(), gomock.Any()).Times(1).Return(*new(error))

			sMock := NewMockfsmContext(ctrl)
			sMock.EXPECT().Finish(true)

			iMock := NewMockinterviewsApi(ctrl)
			repoMock := NewMockrepoClient(ctrl)

			if tt.mock.sender != nil {
				iMock.EXPECT().
					FixTg(gomock.Any(), tt.mock.sender.Username, tt.mock.sender.ID).
					Return((error)(nil))
				iMock.EXPECT().
					FindByUser(gomock.Any(), tt.mock.sender.Username).
					Return(tt.mock.interviews, tt.mock.iErr).
					Times(1)
				repoMock.EXPECT().Interviews().Return(iMock).MaxTimes(2)
			}

			tMock := NewMockTimeProvider(ctrl)
			tMock.EXPECT().ZoneName().Return("").AnyTimes()

			failed := false
			log := zap.NewExample(
				zap.Hooks(func(e zapcore.Entry) error {
					if e.Level > zapcore.InfoLevel {
						failed = true
					}
					return nil
				}),
			).Sugar()

			b := &Bot{
				repo: repoMock,
				time: tMock,
				log:  log,
			}

			err := b.showInterviews(cMock, sMock)
			require.NoError(t, err)
			require.Equal(t, tt.want.fail, failed)
		})
	}
}
