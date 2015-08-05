package go_logger

import (
    "fmt"
    "os"
    "path/filepath"
    "runtime"
    "strconv"
    "strings"
    "sync"
    "syscall"
    "time"
)

type Appender interface {
    GetLayout() string
    Write(msg string)
}

type ConsoleAppender struct {
    layout string
}

func NewConsoleAppender(layout string) *ConsoleAppender {
    return &ConsoleAppender{layout}
}

var console_appender_mutex sync.Mutex

func (ca *ConsoleAppender) GetLayout() string {
    return ca.layout
}

func (*ConsoleAppender) Write(msg string) {
    console_appender_mutex.Lock()
    defer console_appender_mutex.Unlock()
    fmt.Print(msg)
}

type StderrAppender struct {
    layout string
}

var stderr_appender_mutex sync.Mutex

func (ea *StderrAppender) SetLayout(layout string) {
    ea.layout = layout
}

func (ea *StderrAppender) GetLayout() string {
    return ea.layout
}

func (*StderrAppender) Write(msg string) {
    stderr_appender_mutex.Lock()
    defer stderr_appender_mutex.Unlock()
    os.Stderr.WriteString(msg)
}

type FileAppender struct {
    Layout string
    FileName string
    MaxSize int64
    file *os.File
    current_size int64
}

func NewFileAppender(layout, file_name string, max_size int64) *FileAppender {
    return &FileAppender{Layout: layout, FileName: file_name, MaxSize: max_size}
}

func (l *FileAppender) CloseFile() {
    if l.file != nil {
        l.file.Close()
        l.file = nil
    }
    l.current_size = 0
}

func (l *FileAppender) GetLayout() string {
    return l.Layout
}

func (f *FileAppender) get_current_size(file_name string) int64 {
    if f.current_size > 0 {
        return f.current_size
    } else if f.file != nil {
        if fi, err := f.file.Stat(); err == nil {
            f.current_size = fi.Size()
        }
        return f.current_size
    } else {
        if fi, err := os.Lstat(file_name); err == nil {
            f.current_size = fi.Size()
        }
        return f.current_size
    }
}

func (f *FileAppender) open_and_write(file_name, msg string) {
    var err error
    if f.file == nil {
        var path string
        clean := filepath.Clean(file_name)
        if path, err = filepath.Abs(clean); err != nil {
            path = clean
        }
        f.file, err = os.OpenFile(path, syscall.O_WRONLY | syscall.O_CREAT | syscall.O_APPEND, 0644)
    }
    if err != nil {
        f.file = nil
        l := get_private_logger()
        l.Error("打开日志文件", f.FileName, "失败: ", err.Error())
        return
    } else {
        if fi, err := f.file.Stat(); err == nil {
            f.current_size = fi.Size()
        }
    }
    if b, e := f.file.Write([]byte(msg)); e == nil {
        f.current_size += int64(b)
    }
}

func (f *FileAppender) Write(msg string) {
    f.open_and_write(f.FileName, msg)
}

type TruncatedFileAppender struct {
    FileAppender
}

func NewTruncatedFileAppender(layout, file_name string, max_size int64) *TruncatedFileAppender {
    return &TruncatedFileAppender{FileAppender{Layout: layout, FileName: file_name, MaxSize: max_size}}
}

func (f *TruncatedFileAppender) Write(msg string) {
    // 检查是否需要关闭文件
    if f.MaxSize > 0  && (int64(len(msg)) + f.get_current_size(f.FileName) > f.MaxSize) {
        if f.file != nil {
            f.file.Truncate(0)
        } else {
            os.Remove(f.FileName)
        }
    }
    f.open_and_write(f.FileName, msg)
}

type FixSizeFileAppender struct {
    FileAppender
    current_file_name string
    count int
    MaxCount int
}

func NewFixSizeFileAppender(layout, file_name string, max_size int64) *FixSizeFileAppender {
    ret := new(FixSizeFileAppender)
    ret.FileAppender = FileAppender{Layout: layout, FileName: file_name, MaxSize: max_size}
    ret.current_file_name = file_name
    return ret
}

func (f *FixSizeFileAppender) get_current_size(file_name string) int64 {
    if f.current_size > 0 {
        return f.current_size
    } else if f.file != nil {
        if fi, err := f.file.Stat(); err == nil {
            f.current_size = fi.Size()
        }
        return f.current_size
    } else {
        path, name := filepath.Split(f.FileName)
        if path == "" {
            path = filepath.Dir(f.FileName)
        }
        name_dot := name + "."
        filepath.Walk(path, func(fn string, info os.FileInfo, err error) error {
            if strings.HasPrefix(fn, name) {
                if fn == name {
                    f.current_size = info.Size()
                } else {
                    number := strings.TrimPrefix(fn, name_dot)
                    if count, e := strconv.Atoi(number); e == nil && count > f.count {
                        f.count = count
                        f.current_size = info.Size()
                        f.current_file_name = filepath.Join(path, fn)
                    }
                }
            }
            return nil
        })
        if f.MaxCount > 0 && f.count > f.MaxCount {
            f.current_file_name = f.FileName
            f.current_size = 0
            f.count = 0
        }
        return f.current_size
    }
}

func (f *FixSizeFileAppender) Write(msg string) {
    // 检查是否需要关闭文件
    if f.MaxSize > 0 && (int64(len(msg)) + f.get_current_size(f.FileName) > f.MaxSize) {
        if f.file != nil {
            f.CloseFile()
        }
        f.count++
        f.current_file_name = fmt.Sprintf("%s.%d", f.FileName, f.count)
        os.Truncate(f.current_file_name, 0)
        f.current_size = 0
    }
    f.open_and_write(f.current_file_name, msg)
}

type SplittedFileAppender struct {
    FileAppender
    duration time.Duration
    current_file_name string
    next_split_time time.Time
}

func NewSplittedFileAppender(layout, file_name string, duration time.Duration) *SplittedFileAppender {
    return &SplittedFileAppender{FileAppender: FileAppender{Layout: layout, FileName: file_name}, duration: duration, current_file_name: file_name}
}

func (s *SplittedFileAppender) Write(msg string) {
    if new_name, check_time, split := s.should_split(); split {
        s.CloseFile()
        s.current_file_name = new_name
        s.next_split_time = check_time
    }
    s.open_and_write(s.current_file_name, msg)
}

func (s *SplittedFileAppender) should_split() (new_name string, check_time time.Time, split bool) {
    now := time.Now()
    if now.Before(s.next_split_time) {
        split = false
        return
    }
    split = true
    t := now.Truncate(s.duration)
    new_name = fmt.Sprintf("%s.%s", s.FileName, t.Format("20060102.150405"))
    check_time = t.Add(s.duration)
    return
}

type Logger struct {
    name string
    level int
    enabled bool
    time_layout string
    appenders []Appender
    appender_mutex sync.RWMutex
}

const (
    ALL = 0
    TRACE = 10
    DEBUG = 20
    INFO = 30
    LOG = 40
    WARN = 50
    ERROR = 60
    FATAL = 70
    NONE = 100
)

func (l *Logger) SetLevel(level int) *Logger {
    l.level = level
    return l
}

func (l *Logger) enable(on bool) *Logger {
    l.enabled = on
    return l
}

func (l *Logger) SetTimeLayout(layout string) *Logger {
    l.time_layout = layout
    return l
}

func (l *Logger) AddAppender(apd Appender) *Logger {
    l.appender_mutex.Lock()
    defer l.appender_mutex.Unlock()
    l.appenders = append(l.appenders, apd);
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

func (l *Logger) Trace(args... interface{}) *Logger {
    if TRACE >= l.level && l.enabled { l.write("TRACE", args...) }
    return l
}

func (l *Logger) Tracef(layout string, args... interface{}) *Logger {
    if TRACE >= l.level && l.enabled { l.writef("TRACE", layout, args...) }
    return l
}

func (l *Logger) Debug(args... interface{}) *Logger {
    if DEBUG >= l.level && l.enabled { l.write("DEBUG", args...) }
    return l
}

func (l *Logger) Debugf(layout string, args... interface{}) *Logger {
    if DEBUG >= l.level && l.enabled { l.writef("DEBUG", layout, args...) }
    return l
}

func (l *Logger) Info(args... interface{}) *Logger {
    if INFO >= l.level && l.enabled { l.write("INFO", args...) }
    return l
}

func (l *Logger) Infof(layout string, args... interface{}) *Logger {
    if INFO >= l.level && l.enabled { l.writef("INFO", layout, args...) }
    return l
}

func (l *Logger) Log(args... interface{}) *Logger {
    if LOG >= l.level && l.enabled { l.write("LOG", args...) }
    return l
}

func (l *Logger) Logf(layout string, args... interface{}) *Logger {
    if LOG >= l.level && l.enabled { l.writef("LOG", layout, args...) }
    return l
}

func (l *Logger) Warning(args... interface{}) *Logger {
    if WARN >= l.level && l.enabled { l.write("WARNING", args...) }
    return l
}

func (l *Logger) Warningf(layout string, args... interface{}) *Logger {
    if WARN >= l.level && l.enabled { l.writef("WARNING", layout, args...) }
    return l
}

func (l *Logger) Error(args... interface{}) *Logger {
    if ERROR >= l.level && l.enabled { l.write("ERROR", args...) }
    return l
}

func (l *Logger) Errorf(layout string, args... interface{}) *Logger {
    if ERROR >= l.level && l.enabled { l.writef("ERROR", layout, args...) }
    return l
}

func (l *Logger) Fatal(args... interface{}) *Logger {
    if FATAL >= l.level && l.enabled { l.write("FATAL", args...) }
    return l
}

func (l *Logger) Fatalf(layout string, args... interface{}) *Logger {
    if FATAL >= l.level && l.enabled { l.writef("FATAL", layout, args...) }
    return l
}

func (l *Logger) Todo(args... interface{}) *Logger {
    if l.enabled { l.write("TODO", args...) }
    return l
}

func (l *Logger) Todof(layout string, args... interface{}) *Logger {
    if  l.enabled { l.writef("TODO", layout, args...) }
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

func (l *Logger) write(level string, args... interface{}) {
    int_args, str_args := l.get_src_info(level)
    l.appender_mutex.RLock()
    defer l.appender_mutex.RUnlock()
    for _, a := range l.appenders {
        msg := parse_log_layout(int_args, str_args, a.GetLayout(), args...)
        a.Write(msg)
    }
}

func (l *Logger) writef(level string, layout string, args... interface{}) {
    int_args, str_args := l.get_src_info(level)
    l.appender_mutex.RLock()
    defer l.appender_mutex.RUnlock()
    for _, a := range l.appenders {
        msg := parse_log_layout(int_args, str_args, layout, args...)
        a.Write(msg)
    }
}

var global_logger_map map[string] *Logger = make(map[string] *Logger)
var global_map_mutex sync.Mutex

func create_logger(name string) *Logger {
    global_map_mutex.Lock()
    defer global_map_mutex.Unlock()
    ret := global_logger_map[name]
    if ret == nil {
        ret = &Logger{name: name, time_layout : "2006-01-02 15:04:05.999999", appenders : make([]Appender, 0, 10), level: ALL, enabled: true}
        global_logger_map[name] = ret
    }
    return ret
}

func get_time_string(layout string) string {
    t := time.Now()
    return t.Format(layout)
}

func get_private_logger () (l *Logger) {
    l = GetLogger("go_logger")
    if len(l.appenders) == 0 {
        l.AddAppender(&StderrAppender{"[%T] %N-%L: %M"})
    }
    return
}

func parse_log_layout(int_args [2]int, str_args [5]string, layout string, args... interface{}) string {
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

