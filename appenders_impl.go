package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

type ConsoleAppender struct {
	layout Layout
}

var console_appender_mutex sync.Mutex

func (ca *ConsoleAppender) GetLayout() Layout {
	return ca.layout
}

func (*ConsoleAppender) Write(msg string) {
	console_appender_mutex.Lock()
	defer console_appender_mutex.Unlock()
	fmt.Print(msg)
}

type StderrAppender struct {
	layout Layout
}

var stderr_appender_mutex sync.Mutex

/*
func (ea *StderrAppender) SetLayout(layout string) {
	ea.layout = layout
}
*/
func (ea *StderrAppender) GetLayout() Layout {
	return ea.layout
}

func (*StderrAppender) Write(msg string) {
	stderr_appender_mutex.Lock()
	defer stderr_appender_mutex.Unlock()
	os.Stderr.WriteString(msg)
}

type FileAppender struct {
	Layout       string
	FullName     string
	FileName     string
	FileExt      string
	MaxSize      int64
	file         *os.File
	current_size int64
}

func split_file_name(name string) (string, string) {
	if pos := strings.LastIndex(name, "."); pos == -1 {
		return name, ""
	} else {
		use := []byte(name)
		return string(use[:pos]), string(use[pos+1:])
	}
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
		f.file, err = os.OpenFile(path, syscall.O_WRONLY|syscall.O_CREAT|syscall.O_APPEND, 0644)
	}
	if err != nil {
		f.file = nil
		l := get_private_logger()
		l.Error("打开日志文件", f.FullName, "失败: ", err.Error())
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
	f.open_and_write(f.FullName, msg)
}

type TruncatedFileAppender struct {
	FileAppender
}

func (f *TruncatedFileAppender) Write(msg string) {
	// 检查是否需要关闭文件
	if f.MaxSize > 0 && (int64(len(msg))+f.get_current_size(f.FullName) > f.MaxSize) {
		if f.file != nil {
			f.file.Truncate(0)
		} else {
			os.Remove(f.FullName)
		}
	}
	f.open_and_write(f.FullName, msg)
}

type FixSizeFileAppender struct {
	FileAppender
	current_file_name string
	count             int
	MaxCount          int
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
		path, name := filepath.Split(f.FullName)
		if path == "" {
			path = filepath.Dir(f.FullName)
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
			f.current_file_name = f.FullName
			f.current_size = 0
			f.count = 0
		}
		return f.current_size
	}
}

func (f *FixSizeFileAppender) Write(msg string) {
	// 检查是否需要关闭文件
	if f.MaxSize > 0 && (int64(len(msg))+f.get_current_size(f.FullName) > f.MaxSize) {
		if f.file != nil {
			f.CloseFile()
		}
		f.count++
		f.current_file_name = fmt.Sprintf("%s.%d", f.FullName, f.count)
		os.Truncate(f.current_file_name, 0)
		f.current_size = 0
	}
	f.open_and_write(f.current_file_name, msg)
}

type SplittedFileAppender struct {
	FileAppender
	duration          time.Duration
	current_file_name string
	next_split_time   time.Time
}

func (s *SplittedFileAppender) Write(msg string) {
	if new_name, check_time, split := s.should_split(); split {
		s.CloseFile()
		s.current_file_name = new_name
		s.next_split_time = check_time
	}
	s.open_and_write(s.current_file_name, msg)
}

func (s *SplittedFileAppender) should_split_per_day() (new_name string, check_time time.Time, split bool) {
	now := time.Now()
	if now.Before(s.next_split_time) {
		split = false
		return
	}
	split = true
	t, _ := time.Parse("20060102", now.Format("20060102"))
	new_name = fmt.Sprintf("%s.%s.%s", s.FileName, t.Format("20060102"), s.FileExt)
	check_time = t.Add(s.duration)
	return
}

func (s *SplittedFileAppender) should_split() (new_name string, check_time time.Time, split bool) {
	if (s.duration % (24 * time.Hour)) == 0 {
		return s.should_split_per_day()
	}
	now := time.Now()
	if now.Before(s.next_split_time) {
		split = false
		return
	}
	split = true
	t := now.Truncate(s.duration)
	new_name = fmt.Sprintf("%s.%s.%s", s.FileName, t.Format("20060102.150405"), s.FileExt)
	check_time = t.Add(s.duration)
	return
}
