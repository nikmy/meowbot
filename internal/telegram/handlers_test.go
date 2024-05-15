package telegram

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vitaliy-ukiru/fsm-telebot"
	"go.uber.org/mock/gomock"
	"gopkg.in/telebot.v3"

	"github.com/nikmy/meowbot/internal/interviews"
)

func TestBot_showInterviews(t *testing.T) {
	type mocks struct {
		sender *telebot.User

		interviews []interviews.Interview
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
				sender: &telebot.User{Username: "test"},
				iErr:   errors.New("mock"),
			},
			want: want{fail: true},
		},
		{
			name: "no assigned interviews",
			mock: mocks{
				sender: &telebot.User{Username: "test"},
			},
			want: want{fail: false},
		},
		{
			name: "have assigned interviews",
			mock: mocks{
				sender:     &telebot.User{Username: "test"},
				interviews: []interviews.Interview{{}, {}},
			},
			want: want{fail: false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			cMock := NewMocktelebotContext(ctrl)

			cMock.EXPECT().Sender().Return(tt.mock.sender).Times(1)
			cMock.EXPECT().Send(gomock.Any()).Times(1).Return(*new(error))

			sMock := NewMockfsmContext(ctrl)
			sMock.EXPECT().Set(initialState)

			iMock := NewMockinterviewsApi(ctrl)

			if tt.mock.sender != nil {
				iMock.EXPECT().
					FindByUser(gomock.Any(), tt.mock.sender.Username).
					Return(tt.mock.interviews, tt.mock.iErr).
					MaxTimes(1)
			}

			log := NewMockloggerImpl(ctrl)
			if tt.want.fail {
				log.EXPECT().Error(gomock.Any()).Times(1)
			}

			b := &Bot{
				interviews: iMock,
				log:        log,
			}

			err := b.showInterviews(cMock, sMock)
			require.NoError(t, err)
		})
	}
}
