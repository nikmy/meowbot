package txn

import "github.com/nikmy/meowbot/internal/repo"

type Txn interface {
	Start() error
	Abort() error
	Commit() error
}

func New(c repo.Client, model Model) Txn {

}
