package hr

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/gofiber/fiber/v2"

	"github.com/nikmy/meowbot/internal/repo"
	"github.com/nikmy/meowbot/internal/repo/models"
	"github.com/nikmy/meowbot/pkg/errors"
	"github.com/nikmy/meowbot/pkg/logger"
)

func NewServer(cfg Config, log logger.Logger, repoClient repo.Client) Server {
	serveLog := log.With("api_http_server")

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
		serveLog.Warn(errors.WrapFail(err, "handle http request"))
		return c.Status(http.StatusInternalServerError).Send(nil)
	}

	s := &server{
		repo: repoClient,
		http: fiber.New(fiberCfg),
		addr: cfg.HTTP.Addr,
		log:  serveLog,
	}

	s.setupRoutes()

	return s
}

type server struct {
	repo repo.Client
	http *fiber.App
	addr string
	log  logger.Logger
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
	s.http.Patch("/upsertEmployee", s.handleUpsertEmployee)
}

func (s *server) handleUpsertEmployee(c *fiber.Ctx) error {
	var req struct {
		TG string `json:"tg"`
		HR bool   `json:"hr"`
	}

	err := json.Unmarshal(c.Body(), &req)
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

	return nil
}
