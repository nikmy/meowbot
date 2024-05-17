package interviews

import (
	"context"
)

type API interface {
	Txn(ctx context.Context, do func() error) (bool, error)

	// Create is API method for registering an interview. Data may contain confidential information.
	Create(ctx context.Context, vacancy string, candidateTg int64) (id string, err error)

	Delete(ctx context.Context, id string) (found bool, err error)

	Schedule(ctx context.Context, id string, interviewerTg int64, timeSlot [2]int64) error

	Notify(ctx context.Context, id string, at int64, notified []Role) error

	// Find checks whether an interview has been created or not
	Find(ctx context.Context, id string) (*Interview, error)

	// FindByUser returns all user's interviews
	FindByUser(ctx context.Context, user string) ([]Interview, error)

	// GetReadyAt returns list of interviews that have started and not finished at the given timestamp.
	GetReadyAt(ctx context.Context, at int64) (interviews []Interview, err error)

	// Cancel cancels the interview, making it done without results.
	Cancel(ctx context.Context, id string, side Role) (err error)

	// Done marks the interview done. Some logic can be added for sending results template to be filling in
	// to the interviewer, or something else.
	Done(ctx context.Context, id string) (err error)

	Close(ctx context.Context) error
}

type Interview struct {
	ID            string `json:"id"          bson:"_id,omitempty"`
	InterviewerTg int64  `json:"interviewer" bson:"interviewer"`
	CandidateTg   int64  `json:"candidate"   bson:"candidate"`

	Vacancy string `json:"vacancy"     bson:"vacancy"`
	Data    []byte `json:"data"        bson:"data"`

	Interval [2]int64        `json:"interval" bson:"interval"`
	Status   InterviewStatus `json:"status"    bson:"status"`

	CancelledBy Role `json:"cancelled_by" bson:"cancelled_by"`

	LastNotification *NotificationLog `json:"last_notifications" bson:"last_notifications"`
}

type NotificationLog struct {
	UnixTime int64   `json:"unix_time" bson:"unix_time"`
	Notified [2]bool `json:"notified" bson:"notified"`
}

type InterviewStatus int

type Role int

const (
	RoleInterviewer Role = iota
	RoleCandidate
)

const (
	// StatusNew is set when interview has been created
	StatusNew = InterviewStatus(iota) + 1

	// StatusScheduled is set when its tine is known
	StatusScheduled

	// StatusFinished is set when the interview is done
	StatusFinished

	// StatusCancelled is set when it has been cancelled
	StatusCancelled
)
