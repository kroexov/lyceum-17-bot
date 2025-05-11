package app

import (
	"botsrv/pkg/botsrv"
	"context"
	"github.com/go-telegram/bot/models"
	"io"
	"net/http"
	"time"

	"botsrv/pkg/db"
	"botsrv/pkg/embedlog"

	"github.com/go-pg/pg/v10"
	"github.com/go-telegram/bot"
	"github.com/labstack/echo/v4"
	"github.com/vmkteam/zenrpc/v2"
)

type Config struct {
	Database *pg.Options
	Server   struct {
		Host      string
		Port      int
		IsDevel   bool
		EnableVFS bool
	}
	Bot botsrv.Config
}

type App struct {
	embedlog.Logger
	appName string
	cfg     Config
	db      db.DB
	dbc     *pg.DB
	echo    *echo.Echo
	vtsrv   zenrpc.Server

	b  *bot.Bot
	bm *botsrv.BotManager
}

func New(appName string, verbose bool, cfg Config, db db.DB, dbc *pg.DB) *App {
	a := &App{
		appName: appName,
		cfg:     cfg,
		db:      db,
		dbc:     dbc,
		echo:    echo.New(),
	}
	a.SetStdLoggers(verbose)
	a.echo.HideBanner = true
	a.echo.HidePort = true
	a.echo.IPExtractor = echo.ExtractIPFromRealIPHeader()

	a.bm = botsrv.NewBotManager(a.Logger, a.db, a.cfg.Bot)

	opts := []bot.Option{bot.WithDefaultHandler(a.bm.PrivateOnly(a.bm.DefaultHandler))}
	b, err := bot.New(cfg.Bot.Token, opts...)
	if err != nil {
		panic(err)
	}
	a.b = b

	return a
}

// Run is a function that runs application.
func (a *App) Run() error {
	a.registerMetrics()
	a.registerHandlers()
	a.registerDebugHandlers()
	a.registerAPIHandlers()

	a.bm.RegisterBotHandlers(a.b)
	go a.b.Start(context.TODO())
	return a.runHTTPServer(a.cfg.Server.Host, a.cfg.Server.Port)
}

// Shutdown is a function that gracefully stops HTTP server.
func (a *App) Shutdown(timeout time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := a.echo.Shutdown(ctx); err != nil {
		a.Errorf("shutting down server err=%q", err)
	}
}

func (a *App) handleFormResult(c echo.Context) error {
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		a.Errorf("%v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Ошибка чтения тела запроса",
		})
	}

	update := &models.Update{
		CallbackQuery: &models.CallbackQuery{
			Data: string(body),
		},
	}

	switch c.Path() {
	case RouteSubmitStudentForm:
		a.bm.ModerationStudent(c.Request().Context(), a.b, update)
	case RouteSubmitGraduateForm:
		a.bm.ModerationGraduate(c.Request().Context(), a.b, update)
	default:
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Неизвестный путь",
		})
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "Данные переданы на модерацию"})
}
