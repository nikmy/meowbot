package hr

import (
	"context"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"github.com/nikmy/meowbot/internal/repo"
	"github.com/nikmy/meowbot/internal/repo/models"
	"github.com/nikmy/meowbot/pkg/errors"
)

func NewServer(
	cfg Config,
	log *zap.SugaredLogger,
	repoClient repo.Client,
	reqIdGetter reqIdGetter,
	auth authorizer,
) Server {
	serveLog := log.Named("api_http_server")

	fiberCfg := fiber.Config{
		ReadTimeout:             cfg.HTTP.ReadTimeout,
		WriteTimeout:            cfg.HTTP.WriteTimeout,
		IdleTimeout:             cfg.HTTP.IdleTimeout,
		DisableStartupMessage:   true,
		StreamRequestBody:       true,
		EnableTrustedProxyCheck: true,
		ProxyHeader:             cfg.Proxy.Header,
		TrustedProxies:          cfg.Proxy.Trusted,
		RequestMethods:          []string{fiber.MethodGet, fiber.MethodPost},
	}

	fiberCfg.ErrorHandler = func(c *fiber.Ctx, err error) error {
		reqID := reqIdGetter.GetRequestId(c.Request())
		serveLog.With(
			zap.String("request_id", reqID),
			zap.Any("body", string(c.Body())),
		).Error(err)
		return c.Status(http.StatusInternalServerError).Send(nil)
	}

	s := &server{
		repo: repoClient,
		http: fiber.New(fiberCfg),
		addr: cfg.HTTP.Addr,
		auth: auth,
		log:  serveLog,
	}

	s.setupRoutes()

	return s
}

type server struct {
	repo repo.Client
	http *fiber.App
	addr string
	auth authorizer
	log  *zap.SugaredLogger
}

func (s *server) Serve(ctx context.Context) error {
	errCh := make(chan error)
	go func() { errCh <- s.http.Listen(s.addr) }()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return errors.Error("serve context done")
	}
}

func (s *server) Shutdown(ctx context.Context) error {
	var errs []error
	err := s.repo.Close(ctx)
	if err != nil {
		errs = append(errs, errors.WrapFail(err, "close repo"))
	}

	err = s.http.ShutdownWithContext(ctx)
	if err != nil {
		errs = append(errs, errors.WrapFail(err, "shutdown http server"))
	}

	return errors.Join(errs...)
}

func (s *server) setupRoutes() {
	s.http.Post("/upsertEmployee", s.authWrapper(s.handleUpsertEmployee))
	s.http.Post("/interviewData", s.authWrapper(s.handleInterviewData))
}

func (s *server) authWrapper(h fiber.Handler) fiber.Handler {
	if s.auth == nil {
		return h
	}

	return func(c *fiber.Ctx) error {
		ok, err := s.auth.Authorize(c.Request())
		if err != nil {
			return errors.WrapFail(err, "authorize")
		}

		if !ok {
			return c.Status(http.StatusUnauthorized).Send(nil)
		}

		return h(c)
	}
}

func (s *server) handleInterviewData(c *fiber.Ctx) error {
	iid := c.Query("iid", "")
	if iid == "" {
		return c.Status(http.StatusBadRequest).
			Send([]byte("{\"error\": \"interview id param \\\"iid\\\"\" must be provided}"))
	}

	var patch struct {
		Vacancy   *string `json:"vacancy"`
		Candidate *string `json:"candidate"`
		Data      *[]byte `json:"data"`
		Zoom      *string `json:"zoom"`
	}
	err := c.JSON(&patch)
	if err != nil {
		return errors.WrapFail(err, "unmarshal patch data")
	}

	err = s.repo.Interviews().Update(c.Context(), iid, patch.Vacancy, patch.Candidate, patch.Data, patch.Zoom)
	if err != nil {
		return errors.WrapFail(err, "do Interviews.Find request")
	}

	return c.Status(http.StatusOK).Send(nil)
}

func (s *server) handleUpsertEmployee(c *fiber.Ctx) error {
	var req struct {
		TG string `json:"tg"`
		HR bool   `json:"hr"`
	}

	err := c.JSON(&req)
	if err != nil {
		return errors.WrapFail(err, "unmarshal body as json")
	}

	cat := models.EmployeeUser
	if req.HR {
		cat = models.HRUser
	}

	_, err = s.repo.Users().Upsert(c.Context(), req.TG, nil, &cat, nil)
	if err != nil {
		return errors.WrapFail(err, "do Users.Upsert request")
	}

	return c.Status(http.StatusOK).Send(nil)

}
