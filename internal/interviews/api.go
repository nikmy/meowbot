package interviews

import "context"

type API interface {
	// Create is API method for registering an interview. Data may contain confidential information.
	Create(ctx context.Context, data any, interviewerTg string, candidateTg string) (id string, err error)

	// Propose is API method for candidate which is used for asking the interviewer whether he/she
	// can do interview in the proposed time intervals. Time must be unix timestamp UTC.
	Propose(ctx context.Context, id string, intervals [][2]uint64) (err error)

	// Accept and Decline are API methods for interviewer which is used for responding to interview
	// time request sent by candidate. Accepted interval may be subinterval of one of proposed.
	Accept(ctx context.Context, id string, interval [2]uint64) (err error)
	Decline(ctx context.Context, id string) (err error)

	// GetReadyAt returns list of interviews that have started and not finished at the given timestamp.
	GetReadyAt(ctx context.Context, at uint64) (interviews []Interview, err error)

	// Cancel cancels the interview, making it done without results.
	Cancel(ctx context.Context, id string, reason string) (err error)

	// Done marks the interview done. Some logic can be added for sending results template to be fiiling in
	// to the interviewer, or something else.
	Done(ctx context.Context, id string) (err error)
}

type Interview struct {
	ID            string `json:"id"          bson:"-"`
	InterviewerTg string `json:"interviewer" bson:"interviewer"`
	CandidateTg   string `json:"candidate"   bson:"candidate"`
	Data          any    `json:"data"        bson:"data"`

	Intervals [][2]uint64     `json:"intervals" bson:"intervals"`
	Status    InterviewStatus `json:"status"    bson:"status"`

	CancelReason string `json:"cancel_reason" bson:"cancel_reason"`
}

type InterviewStatus int

const (
	// StatusNew is set when interview has been created
	StatusNew = InterviewStatus(iota) + 1

	// StatusAsk is set when candidate proposed intervals
	StatusAsk

	// StatusAccepted is set when interviewer accepted interval
	StatusAccepted

	// StatusDeclined is set when interviewer declined candidate's intervals
	StatusDeclined

	// StatusFinished is set when the interview is done
	StatusFinished

	// StatusCancelled is set when it has been cancelled
	StatusCancelled
)

func (i Interview) GetID() string {
	return i.ID
}

func (i Interview) SetID(id string) {
	i.ID = id
}
