package models

import "context"

type InterviewsRepo interface {
	// Create is API method for registering an interview. Data may contain confidential information.
	Create(ctx context.Context, vacancy string, candidateTg string) (id string, err error)

	// Delete completely removes interview object
	Delete(ctx context.Context, id string) (found *Interview, err error)

	// Update patches interview
	Update(ctx context.Context, id string, vacancy *string, candidate *string, data *[]byte, zoom *string) error

	// Schedule assigns interview to interviewer
	Schedule(ctx context.Context, id string, candidate User, interviewer User, slot Meeting) error

	// Notify saves information about notification
	Notify(ctx context.Context, id string, at int64, notified [2]bool) error

	// Find checks whether an interview has been created or not
	Find(ctx context.Context, id string) (*Interview, error)

	// FindByUser returns all user's interviews
	FindByUser(ctx context.Context, username string) ([]*Interview, error)

	// GetUpcoming returns list of upcoming interviews of fixed (1024) size.
	GetUpcoming(ctx context.Context, lastNotifyBefore, startsBefore int64) (interviews []*Interview, err error)

	// Cancel cancels the interview, making it done without results.
	Cancel(ctx context.Context, id string, side Role) (err error)

	// Done marks the interview done. Some logic can be added for sending results template to be filling in
	// to the interviewer, or something else.
	Done(ctx context.Context, id string) (err error)

	// FixTg sets candidateTg value for interviews with candidate == username
	FixTg(ctx context.Context, username string, tg int64) (err error)
}

type Interview struct {
	ID      string `json:"id"          bson:"_id,omitempty"`
	Vacancy string `json:"vacancy"     bson:"vacancy"`

	CandidateUN   string `json:"candidate"   bson:"candidate"`
	InterviewerUN string `json:"interviewer" bson:"interviewer"`

	CandidateTg   int64 `json:"candidate_tg"   bson:"candidate_tg"`
	InterviewerTg int64 `json:"interviewer_tg" bson:"interviewer_tg"`

	Data []byte `json:"data"        bson:"data"`
	Zoom string `json:"zoom"        bson:"zoom"`

	Status      InterviewStatus `json:"status"       bson:"status"`
	Meet        *[2]int64       `json:"meet"         bson:"meet"`
	CancelledBy Role            `json:"cancelled_by" bson:"cancelled_by"`

	LastNotification *NotificationLog `json:"last_notification" bson:"last_notification"`
}

const (
	InterviewFieldID               = "id"
	InterviewFieldCandidateUN      = "candidate"
	InterviewFieldInterviewerUN    = "interviewer"
	InterviewFieldCandidateTg      = "candidate_tg"
	InterviewFieldInterviewerTg    = "interviewer_tg"
	InterviewFieldVacancy          = "vacancy"
	InterviewFieldData             = "data"
	InterviewFieldZoom             = "zoom"
	InterviewFieldMeet             = "meet"
	InterviewFieldStatus           = "status"
	InterviewFieldCancelledBy      = "cancelled_by"
	InterviewFieldLastNotification = "last_notification"
)

type NotificationLog struct {
	UnixTime int64   `json:"unix_time" bson:"unix_time"`
	Notified [2]bool `json:"notified" bson:"notified"`
}

const (
	NotificationFieldUnixTime = "unix_time"
	NotificationFieldNotified = "notified"
)

type InterviewStatus int

type Role int

const (
	RoleInterviewer Role = iota
	RoleCandidate
	RoleHR
)

const (
	// InterviewStatusNew is set when interview has been created
	InterviewStatusNew = InterviewStatus(iota)

	// InterviewStatusScheduled is set when its tine is known
	InterviewStatusScheduled

	// InterviewStatusFinished is set when the interview is done
	InterviewStatusFinished

	// InterviewStatusCancelled is set when it has been cancelled
	InterviewStatusCancelled
)
