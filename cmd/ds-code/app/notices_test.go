package app

import (
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/logging"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/header"
)

func TestBuildStartupNotices_sensitiveLog(t *testing.T) {
	a := &App{Cfg: &config.Config{AllowLogSensitiveData: true}}
	notices := buildStartupNotices(a)
	if len(notices) != 1 {
		t.Fatalf("notices = %d", len(notices))
	}
	if notices[0].Level != header.NoticeWarn || notices[0].Text != logging.SensitiveDataWarningMsg {
		t.Fatalf("notice = %+v", notices[0])
	}
}

func TestBuildStartupNotices_empty(t *testing.T) {
	a := &App{Cfg: &config.Config{}}
	if got := buildStartupNotices(a); len(got) != 0 {
		t.Fatalf("notices = %+v", got)
	}
}
