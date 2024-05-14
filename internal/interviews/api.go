package interviews

import "context"

type API interface {
	// Create is API method for registering an interview. Data may contain confidential information.
	Create(ctx context.Context, vacancy string, candidateTg string) (id string, err error)

	Delete(ctx context.Context, id string) (found bool, err error)

	Schedule(ctx context.Context, id string, interviewer string, timeSlot [2]int64) error

	// Find checks whether an interview has been created or not
	Find(ctx context.Context, id string) (*Interview, error)

	FindByCandidate(ctx context.Context, candidate string) ([]Interview, error)

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
	ID            string `json:"id"          bson:"-"`
	InterviewerTg string `json:"interviewer" bson:"interviewer"`
	CandidateTg   string `json:"candidate"   bson:"candidate"`

	Vacancy string `json:"info" bson:"info"`
	Data    []byte `json:"data"        bson:"data"`

	Interval [2]int64        `json:"intervals" bson:"intervals"`
	Status   InterviewStatus `json:"status"    bson:"status"`

	CancelledBy Role `json:"cancelled_by" bson:"cancelled_by"`
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
