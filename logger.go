package logger

import (
	"bytes"
	"fmt"
	"runtime"
	"sync"
)

const (
	ALL   = 0
	TRACE = 10
	DEBUG = 20
	INFO  = 30
	LOG   = 40
	WARN  = 50
	ERROR = 60
	FATAL = 70
	NONE  = 100
)

type (
	Logger struct {
		ori_name       string
		name           string
		level          int
		enabled        bool
		time_layout    string
		appenders      []Appender
		appender_mutex sync.RWMutex
		skip_pc        int
	}
)

func CallStack(skip ...int) string {
	s := 2
	if len(skip) > 0 {
		s = skip[0]
	}
	pc := make([]uintptr, 100)
	n := runtime.Callers(s, pc)
	if n == 0 {
		return ""
	}
	pc = pc[:n]
	frames := runtime.CallersFrames(pc)
	buf := bytes.NewBufferString("")
	for frame, more := frames.Next(); more; frame, more = frames.Next() {
		buf.WriteString(fmt.Sprintf("%s:%d - %s\n", frame.File, frame.Line, frame.Function))
	}
	return buf.String()
}

func (l *Logger) SetName(name string) *Logger {
	l.name = name
	return l
}

func (l *Logger) SetLevel(level int) *Logger {
	l.level = level
	return l
}

func (l *Logger) Enable(on bool) *Logger {
	l.enabled = on
	return l
}

func (l *Logger) On() *Logger {
	l.enabled = true
	return l
}

func (l *Logger) Off() *Logger {
	l.enabled = false
	return l
}

func (l *Logger) SetTimeLayout(layout string) *Logger {
	l.time_layout = layout
	return l
}

func (l *Logger) AddAppender(apd Appender) *Logger {
	l.appender_mutex.Lock()
	defer l.appender_mutex.Unlock()
	l.appenders = append(l.appenders, apd)
	return l
}

func (l *Logger) ClearAppender() *Logger {
	l.appender_mutex.Lock()
	defer l.appender_mutex.Unlock()
	l.appenders = make([]Appender, 0, 10)
	return l
}

func (l *Logger) AppenderCount() int {
	return len(l.appenders)
}

func (l *Logger) SkipPC(v int) *Logger {
	l.skip_pc = v
	return l
}

func (l *Logger) Trace(args ...interface{}) *Logger {
	if TRACE >= l.level && l.enabled {
		l.write("TRACE", args...)
	}
	return l
}

func (l *Logger) Tracef(layout string, args ...interface{}) *Logger {
	if TRACE >= l.level && l.enabled {
		l.writef("TRACE", layout, args...)
	}
	return l
}

func (l *Logger) Debug(args ...interface{}) *Logger {
	if DEBUG >= l.level && l.enabled {
		l.write("DEBUG", args...)
	}
	return l
}

func (l *Logger) Debugf(layout string, args ...interface{}) *Logger {
	if DEBUG >= l.level && l.enabled {
		l.writef("DEBUG", layout, args...)
	}
	return l
}

func (l *Logger) Info(args ...interface{}) *Logger {
	if INFO >= l.level && l.enabled {
		l.write("INFO", args...)
	}
	return l
}

func (l *Logger) Infof(layout string, args ...interface{}) *Logger {
	if INFO >= l.level && l.enabled {
		l.writef("INFO", layout, args...)
	}
	return l
}

func (l *Logger) Log(args ...interface{}) *Logger {
	if LOG >= l.level && l.enabled {
		l.write("LOG", args...)
	}
	return l
}

func (l *Logger) Logf(layout string, args ...interface{}) *Logger {
	if LOG >= l.level && l.enabled {
		l.writef("LOG", layout, args...)
	}
	return l
}

func (l *Logger) Warning(args ...interface{}) *Logger {
	if WARN >= l.level && l.enabled {
		l.write("WARNING", args...)
	}
	return l
}

func (l *Logger) Warningf(layout string, args ...interface{}) *Logger {
	if WARN >= l.level && l.enabled {
		l.writef("WARNING", layout, args...)
	}
	return l
}

func (l *Logger) Error(args ...interface{}) *Logger {
	if ERROR >= l.level && l.enabled {
		l.write("ERROR", args...)
	}
	return l
}

func (l *Logger) Errorf(layout string, args ...interface{}) *Logger {
	if ERROR >= l.level && l.enabled {
		l.writef("ERROR", layout, args...)
	}
	return l
}

func (l *Logger) Fatal(args ...interface{}) *Logger {
	if FATAL >= l.level && l.enabled {
		l.write("FATAL", args...)
	}
	return l
}

func (l *Logger) Fatalf(layout string, args ...interface{}) *Logger {
	if FATAL >= l.level && l.enabled {
		l.writef("FATAL", layout, args...)
	}
	return l
}

func (l *Logger) Todo(args ...interface{}) *Logger {
	if l.enabled {
		l.write("TODO", args...)
	}
	return l
}

func (l *Logger) Todof(layout string, args ...interface{}) *Logger {
	if l.enabled {
		l.writef("TODO", layout, args...)
	}
	return l
}

func GetLogger(name string) *Logger {
	if ret, exist := get_logger(name); exist {
		return ret
	} else {
		return create_logger(name)
	}
}
