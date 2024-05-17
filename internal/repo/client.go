package repo

import "github.com/nikmy/meowbot/internal/repo/models"

type Client interface {
	Interviews() models.InterviewsRepo
	Users() models.UsersRepo

	RunTxn(fn func(c Client))
}

type Table interface {

}
