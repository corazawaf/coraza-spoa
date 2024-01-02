package server

import (
	"fmt"
	"strings"
	"time"

	"github.com/corazawaf/coraza-spoa/internal/cache"
	"github.com/corazawaf/coraza-spoa/internal/config"
	"github.com/corazawaf/coraza-spoa/internal/logger"
	"github.com/corazawaf/coraza/v3"
	"github.com/corazawaf/coraza/v3/types"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type ErrInterrupted struct {
	Interruption *types.Interruption
}

func (e ErrInterrupted) Error() string {
	return fmt.Sprintf("interrupted with status %d and action %s", e.Interruption.Status, e.Interruption.Action)
}

func (e *ErrInterrupted) Is(target error) bool {
	t, ok := target.(*ErrInterrupted)
	if !ok {
		return false
	}
	return e.Interruption == t.Interruption
}

type application struct {
	config.Application
	waf    coraza.WAF
	logger *zerolog.Logger
}

func (a *application) HandleRequest(req *applicationRequest) error {
	id := req.ID
	if id == "" {
		return fmt.Errorf("request id is empty")
	}
	tx := a.waf.NewTransactionWithID(id)
	tx.ProcessConnection(req.SrcIp.String(), int(req.SrcPort), req.DstIp.String(), int(req.DstPort))
	url := strings.Builder{}
	url.WriteString(req.Path)
	if req.Query != "" {
		url.Grow(len(req.Query) + 1)
		url.WriteString("?")
		url.WriteString(req.Query)
	}
	tx.ProcessURI(url.String(), req.Method, "HTTP/"+req.Version)
	if err := readHeaders(req.Headers, func(key, value string) {
		tx.AddRequestHeader(key, value)
	}); err != nil {
		return err
	}
	if it := tx.ProcessRequestHeaders(); it != nil {
		return ErrInterrupted{it}
	}
	if it, _, err := tx.WriteRequestBody(req.Body); it != nil {
		return ErrInterrupted{it}
	} else if err != nil {
		return err
	}
	if it, err := tx.ProcessRequestBody(); it != nil {
		return ErrInterrupted{it}
	} else if err != nil {
		return err
	}
	if a.logger.GetLevel() == zerolog.DebugLevel {
		a.logger.Debug().Msg(req.String())
	}
	if !a.ResponseCheck {
		tx.ProcessLogging()
		tx.Close()
		return nil
	} else {
		cache.Add(tx, time.Duration(a.TransactionTTLMs*time.Millisecond))
	}

	return nil
}

func (a *application) HandleResponse(res *applicationResponse) error {
	id := res.ID
	if id == "" {
		return fmt.Errorf("response id is empty")
	}
	tx, ok := cache.Get(id)
	if !ok {
		return fmt.Errorf("transaction %s not found", id)
	}
	if err := readHeaders(res.Headers, func(key, value string) {
		tx.AddResponseHeader(key, value)
	}); err != nil {
		return err
	}
	if it := tx.ProcessResponseHeaders(int(res.Status), "HTTP/"+res.Version); it != nil {
		return ErrInterrupted{it}
	}
	tx.WriteResponseBody(res.Body)
	if it, err := tx.ProcessResponseBody(); it != nil {
		return ErrInterrupted{it}
	} else if err != nil {
		return err
	}
	tx.ProcessLogging()
	tx.Close()
	// TODO does remove forces eviction?
	cache.Remove(id)

	return nil
}

func (a *application) logCallback(mr types.MatchedRule) {
	var l *zerolog.Event

	switch mr.Rule().Severity() {
	case types.RuleSeverityWarning:
		l = a.logger.Warn()
	case types.RuleSeverityNotice:
		l = a.logger.Info()
	case types.RuleSeverityInfo:
		l = a.logger.Info()
	case types.RuleSeverityDebug:
		l = a.logger.Debug()
	default:
		l = a.logger.Error()
	}
	l.Msg(mr.ErrorLog())

}

func newApplication(app *config.Application) (*application, error) {
	// if no log settings are used, we inherit global settings
	tmpLogger := getApps().logger
	if tmpLogger == nil {
		tmpLogger = &log.Logger
	}
	a := &application{
		Application: *app,
		logger:      tmpLogger,
	}

	if app.LogFile != "" || app.LogLevel != "" {
		logger, err := logger.New(app.LogLevel, app.LogFile)
		if err != nil {
			return nil, err
		}
		logger.Debug().Str("level", app.LogLevel).Str("app", app.Name).
			Str("file", app.LogFile).Msg("application logger created")
		a.logger = logger
	} else {
		a.logger.Info().Str("app", app.Name).Msg("application is inheriting parent logger")
	}
	config := coraza.NewWAFConfig().
		WithDirectives(app.Directives).
		WithErrorCallback(a.logCallback) //.WithRootFS(merged_fs.NewMergedFS(coreruleset.FS, io.OSFS))
		// TODO for some reason it is failing with the merged fs
	a.logger.Debug().Str("app", app.Name).Msg("WAF config created")
	waf, err := coraza.NewWAF(config)
	if err != nil {
		return nil, err
	}
	a.waf = waf
	return a, nil
}
