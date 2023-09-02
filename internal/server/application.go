package server

import (
	"fmt"
	"os"
	"strings"

	coreruleset "github.com/corazawaf/coraza-coreruleset"
	"github.com/corazawaf/coraza-coreruleset/io"
	"github.com/corazawaf/coraza-spoa/internal/config"
	"github.com/corazawaf/coraza/v3"
	"github.com/corazawaf/coraza/v3/types"
	"github.com/negasus/haproxy-spoe-go/message"
	"github.com/rs/zerolog"
	"github.com/yalue/merged_fs"
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
	responseEnabled bool
	waf             coraza.WAF
	logger          zerolog.Logger
}

func (a *application) HandleRequest(id string, req *applicationRequest) error {

	tx := a.waf.NewTransactionWithID(id)
	tx.ProcessConnection(req.srcIp.String(), int(req.srcPort), req.dstIp.String(), int(req.dstPort))
	url := strings.Builder{}
	url.WriteString(req.path)
	if req.query != "" {
		url.Grow(len(req.query) + 1)
		url.WriteString("?")
		url.WriteString(req.query)
	}
	tx.ProcessURI(url.String(), req.method, "HTTP/"+req.version)
	if err := readHeaders(req.headers, func(key, value string) {
		tx.AddRequestHeader(key, value)
	}); err != nil {
		return err
	}
	if it := tx.ProcessRequestHeaders(); it != nil {
		return ErrInterrupted{it}
	}

	if it, err := tx.ProcessRequestBody(); it != nil {
		return ErrInterrupted{it}
	} else if err != nil {
		return err
	}
	if a.logger.GetLevel() == zerolog.DebugLevel {
		a.logger.Debug().Msg(req.String())
	}

	return nil
}

func (a *application) HandleResponse(id string, msg *message.Message) error {

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
	a := &application{
		Application: *app,
		logger:      zerolog.New(os.Stdout),
	}
	config := coraza.NewWAFConfig().
		WithDirectives(app.Directives).
		WithErrorCallback(a.logCallback).
		WithRootFS(merged_fs.NewMergedFS(coreruleset.FS, io.OSFS))
	waf, err := coraza.NewWAF(config)
	if err != nil {
		return nil, err
	}
	a.waf = waf
	return a, nil
}
