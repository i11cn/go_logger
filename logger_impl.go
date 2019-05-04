package logger

import (
	"os"
	"runtime"
	"sync"
	"time"
)

var (
	global_logger_map map[string]*Logger = make(map[string]*Logger)
	global_map_mutex  sync.RWMutex
)

func (l *Logger) get_src_info(level string) (int_args [2]int, str_args [5]string) {
	int_args = [2]int{0, os.Getpid()}
	str_args = [5]string{"", "", l.name, level, get_time_string(l.time_layout)}
	if pc, file, line, ok := runtime.Caller(3); ok {
		f := runtime.FuncForPC(pc)
		int_args[0] = line
		str_args[0], str_args[1] = file, f.Name()
	}
	return
}

func (l *Logger) write(level string, args ...interface{}) {
	int_args, str_args := l.get_src_info(level)
	l.appender_mutex.RLock()
	defer l.appender_mutex.RUnlock()
	for _, a := range l.appenders {
		msg := a.Format(int_args, str_args, args...)
		a.Write(msg)
	}
}

func (l *Logger) writef(level string, layout string, args ...interface{}) {
	int_args, str_args := l.get_src_info(level)
	l.appender_mutex.RLock()
	defer l.appender_mutex.RUnlock()
	lo := Layout{ParseLayout(layout)}
	for _, a := range l.appenders {
		msg := a.FormatBy(lo, int_args, str_args, args...)
		a.Write(msg)
	}
}

func get_logger(name string) (ret *Logger, exist bool) {
	global_map_mutex.RLock()
	defer global_map_mutex.RUnlock()
	ret, exist = global_logger_map[name]
	return
}

func create_logger(name string) *Logger {
	global_map_mutex.Lock()
	defer global_map_mutex.Unlock()
	if ret, exist := global_logger_map[name]; exist {
		return ret
	} else {
		ret = &Logger{ori_name: name, name: name, time_layout: "2006-01-02 15:04:05.000000", appenders: make([]Appender, 0, 10), level: ALL, enabled: true}
		global_logger_map[name] = ret
		return ret
	}
}

func get_time_string(layout string) string {
	return time.Now().Format(layout)
}

func get_private_logger() (l *Logger) {
	l = GetLogger("go_logger")
	if len(l.appenders) == 0 {
		l.AddAppender(&StderrAppender{NewBaseAppender("[%T] %N-%L: %M")})
	}
	return
}
