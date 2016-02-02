package logger

import (
	"fmt"
	"os"
	"runtime"
	"sync"
	"time"
)

type Logger struct {
	name           string
	level          int
	enabled        bool
	time_layout    string
	appenders      []Appender
	appender_mutex sync.RWMutex
}

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
	ret := global_logger_map[name]
	if ret != nil {
		return ret
	}
	return create_logger(name)
}

func (l *Logger) get_src_info(level string) (int_args [2]int, str_args [5]string) {
	if pc, _, _, ok := runtime.Caller(3); ok {
		f := runtime.FuncForPC(pc)
		file, line := f.FileLine(pc)
		int_args = [2]int{line, os.Getpid()}
		str_args = [5]string{file, f.Name(), l.name, level, get_time_string(l.time_layout)}
	} else {
		int_args = [2]int{0, os.Getpid()}
		str_args = [5]string{"", "", l.name, level, get_time_string(l.time_layout)}
	}
	return
}

func (l *Logger) write(level string, args ...interface{}) {
	int_args, str_args := l.get_src_info(level)
	l.appender_mutex.RLock()
	defer l.appender_mutex.RUnlock()
	for _, a := range l.appenders {
		msg := parse_log_layout(int_args, str_args, a.GetLayout(), args...)
		a.Write(msg)
	}
}

func (l *Logger) writef(level string, layout string, args ...interface{}) {
	int_args, str_args := l.get_src_info(level)
	l.appender_mutex.RLock()
	defer l.appender_mutex.RUnlock()
	for _, a := range l.appenders {
		msg := parse_log_layout(int_args, str_args, layout, args...)
		a.Write(msg)
	}
}

var global_logger_map map[string]*Logger = make(map[string]*Logger)
var global_map_mutex sync.Mutex

func create_logger(name string) *Logger {
	global_map_mutex.Lock()
	defer global_map_mutex.Unlock()
	ret := global_logger_map[name]
	if ret == nil {
		ret = &Logger{name: name, time_layout: "2006-01-02 15:04:05.999999", appenders: make([]Appender, 0, 10), level: ALL, enabled: true}
		global_logger_map[name] = ret
	}
	return ret
}

func get_time_string(layout string) string {
	t := time.Now()
	return t.Format(layout)
}

func get_private_logger() (l *Logger) {
	l = GetLogger("go_logger")
	if len(l.appenders) == 0 {
		l.AddAppender(&StderrAppender{"[%T] %N-%L: %M"})
	}
	return
}

func parse_log_layout(int_args [2]int, str_args [5]string, layout string, args ...interface{}) string {
	tag := false
	total := len(args)
	current := 0
	msg := make([]rune, 0, 4096)
	for _, c := range layout {
		if tag {
			// 处理%*中的那个*
			switch c {
			case 'F':
				msg = append(msg, []rune(str_args[0])...)

			case 'f':
				msg = append(msg, []rune(str_args[1])...)

			case 'l':
				msg = append(msg, []rune(fmt.Sprint(int_args[0]))...)

			case 'N':
				msg = append(msg, []rune(str_args[2])...)

			case 'L':
				msg = append(msg, []rune(str_args[3])...)

			case 'p':
				msg = append(msg, []rune(fmt.Sprint(int_args[1]))...)

			case 'T':
				msg = append(msg, []rune(str_args[4])...)

			case '%':
				msg = append(msg, '%')

			case 'm':
				if current < total {
					msg = append(msg, []rune(fmt.Sprint(args[current]))...)
					current++
				} else {
					msg = append(msg, '%', 'm')
				}

			case 'M':
				for ; current < total; current++ {
					msg = append(msg, []rune(fmt.Sprint(args[current]))...)
				}
			}
			tag = false
		} else if c == '%' {
			// 遇到了%*中的那个%
			tag = true
		} else {
			// 啥都没有，正常的一个字符，照原样输出
			msg = append(msg, c)
		}
	}
	msg = append(msg, []rune(fmt.Sprintln())...)
	return string(msg)
}
