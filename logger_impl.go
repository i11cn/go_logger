package logger

import (
	"bytes"
	"fmt"
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
		ret = &Logger{name: name, time_layout: "2006-01-02 15:04:05.999999", appenders: make([]Appender, 0, 10), level: ALL, enabled: true}
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
		l.AddAppender(&StderrAppender{"[%T] %N-%L: %M"})
	}
	return
}

func parse_log_layout(int_args [2]int, str_args [5]string, layout string, args ...interface{}) string {
	tag := false
	total := len(args)
	current := 0
	var msg bytes.Buffer
	for _, c := range layout {
		if tag {
			// 处理%*中的那个*
			switch c {
			case 'F':
				msg.Write([]byte(str_args[0]))

			case 'f':
				msg.Write([]byte(str_args[1]))

			case 'l':
				msg.Write([]byte(fmt.Sprint(int_args[0])))

			case 'N':
				msg.Write([]byte(str_args[2]))

			case 'L':
				msg.Write([]byte(str_args[3]))

			case 'p':
				msg.Write([]byte(fmt.Sprint(int_args[1])))

			case 'T':
				msg.Write([]byte(str_args[4]))

			case '%':
				msg.WriteByte('%')

			case 'm':
				if current < total {
					msg.Write([]byte(fmt.Sprint(args[current])))
					current++
				} else {
					msg.Write([]byte("%m"))
				}

			case 'M':
				for ; current < total; current++ {
					msg.Write([]byte(fmt.Sprint(args[current])))
				}
			}
			tag = false
		} else if c == '%' {
			// 遇到了%*中的那个%
			tag = true
		} else {
			// 啥都没有，正常的一个字符，照原样输出
			msg.WriteRune(c)
		}
	}
	msg.WriteByte('\n')
	return msg.String()
}
