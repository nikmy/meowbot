package api

import (
	"context"
	"github.com/gofiber/fiber/v2"
	"github.com/nikmy/meowbot/internal/repo"
	"github.com/nikmy/meowbot/pkg/errors"
	"github.com/nikmy/meowbot/pkg/logger"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func NewServer(cfg Config, log logger.Logger, repo repo.Repo) Server {
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
		repo: repo,
		http: fiber.New(fiberCfg),
		addr: cfg.HTTP.Addr,
		log:  serveLog,
	}

	s.setupRoutes()

	return s
}

type server struct {
	repo repo.Repo
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

	return errors.Collapse(errs)
}

func (s *server) setupRoutes() {
	s.http.Get("/getByID", s.handleGetByID)
	s.http.Post("/getReady", s.handleGetReady)
	s.http.Post("/new", s.handleNew)
	s.http.Patch("/update", s.handleUpdate)
	s.http.Delete("/delete", s.handleDelete)
}

func (s *server) handleGetByID(c *fiber.Ctx) error {
	id, err := s.getIDOrErr(c)
	if err != nil {
		s.log.Warn(err)
		return s.sendError(c, http.StatusBadRequest, "missing required parameter \"id\"")
	}

	reminder, err := s.repo.Get(c.Context(), id)
	if err != nil {
		return errors.WrapFail(err, "get reminder by id")
	}

	return c.Status(http.StatusOK).JSON(reminder)
}

func (s *server) handleGetReady(c *fiber.Ctx) error {
	at, err := s.getAtOrErr(c)
	if err != nil {
		s.log.Warn(err)
		c = c.Status(http.StatusBadRequest)
		return s.sendError(c, http.StatusBadRequest, "bad or missed parameter \"at\"")
	}

	ready, err := s.repo.GetReadyAt(c.Context(), at)
	if err != nil {
		return errors.WrapFail(err, "get ready reminders from repo")
	}

	return c.Status(http.StatusOK).JSON(ready)
}

func (s *server) handleNew(c *fiber.Ctx) error {
	at, err := s.getAtOrErr(c)
	if err != nil {
		s.log.Warn(err)
		return s.sendError(c, http.StatusBadRequest, "bad or missed parameter \"at\"")
	}

	var data any
	err = c.BodyParser(&data)
	if err != nil {
		err = errors.WrapFail(err, "unmarshal reminder payload")
		s.log.Warn(err)
		return s.sendError(c, http.StatusBadRequest, "bad json")
	}

	channels := s.getChannels(c)

	id, err := s.repo.Create(c.Context(), data, at, channels)
	if err != nil {
		return errors.WrapFail(err, "create reminder")
	}

	return c.Status(http.StatusCreated).JSON(map[string]string{"id": id})
}

func (s *server) handleUpdate(c *fiber.Ctx) error {
	id, err := s.getIDOrErr(c)
	if err != nil {
		s.log.Warn(err)
		return s.sendError(c, http.StatusBadRequest, "missing required parameter \"id\"")
	}

	var patch struct {
		Data any        `json:"data"`
		Time *time.Time `json:"time"`
	}

	err = c.BodyParser(&patch)
	if err != nil {
		s.log.Warn(errors.WrapFail(err, "parse update request"))
		return s.sendError(c, http.StatusBadRequest, "bad patch format")
	}

	_, err = s.repo.Update(c.Context(), id, patch.Data, patch.Time)
	if err != nil {
		return errors.WrapFail(err, "update reminder")
	}

	return c.Status(http.StatusOK).Send(nil)
}

func (s *server) handleDelete(c *fiber.Ctx) error {
	id, err := s.getIDOrErr(c)
	if err != nil {
		s.log.Warn(err)
		return s.sendError(c, http.StatusBadRequest, "missing required parameter \"id\"")
	}

	deleted, err := s.repo.Delete(c.Context(), id)
	if err != nil {
		return errors.WrapFail(err, "delete reminder")
	}

	if !deleted {
		return s.sendError(c, http.StatusNotFound, "nothing to delete")
	}

	return c.Status(http.StatusOK).Send(nil)
}

func (s *server) sendError(c *fiber.Ctx, status int, msg string) error {
	return c.Status(status).JSON(map[string]string{"status": "ERROR", "message": msg})
}

func (s *server) getChannels(c *fiber.Ctx) []string {
	separated := c.Query("channels", "default")
	return strings.Split(separated, ",")
}

func (s *server) getIDOrErr(c *fiber.Ctx) (string, error) {
	id := c.Query("id", "")
	if id == "" {
		return "", errors.Error("got empty \"id\" param of getByID request")
	}

	return id, nil
}

func (s *server) getAtOrErr(c *fiber.Ctx) (time.Time, error) {
	atStr := c.Query("at", "")
	if atStr == "" {
		return time.Time{}, errors.Error("got empty \"at\" param of getReady request")
	}

	atUnix, err := strconv.ParseInt(atStr, 10, 64)
	if err != nil {
		return time.Time{}, errors.Errorf("got malformed \"at\" %s", atStr)
	}

	return time.UnixMilli(atUnix), nil
}
