package backend

import (
	"go.uber.org/zap"
)

type loggingLevel int

const (
	DEBUG loggingLevel = iota
	INFO
	WARNING
	ERROR
)

type zapLog struct {
	*zap.Logger
	level loggingLevel
}

func (l *zapLog) Errorf(f string, v ...interface{}) {
	if l.level <= ERROR {
		l.Errorf("ERROR: "+f, v...)
	}
}

func (l *zapLog) Warningf(f string, v ...interface{}) {
	if l.level <= WARNING {
		l.Warningf("WARNING: "+f, v...)
	}
}

func (l *zapLog) Infof(f string, v ...interface{}) {
	if l.level <= INFO {
		l.Infof("INFO: "+f, v...)
	}
}

func (l *zapLog) Debugf(f string, v ...interface{}) {
	if l.level <= DEBUG {
		l.Debugf("DEBUG: "+f, v...)
	}
}
