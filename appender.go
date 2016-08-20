package logger

import (
	"time"
)

type Appender interface {
	GetLayout() string
	Write(msg string)
}

func NewConsoleAppender(layout string) *ConsoleAppender {
	return &ConsoleAppender{layout}
}

func NewFileAppender(layout, file_name string, max_size int64) *FileAppender {
	name, ext := split_file_name(file_name)
	return &FileAppender{Layout: layout, FullName: file_name, FileName: name, FileExt: ext, MaxSize: max_size}
}

func NewTruncatedFileAppender(layout, file_name string, max_size int64) *TruncatedFileAppender {
	name, ext := split_file_name(file_name)
	return &TruncatedFileAppender{FileAppender{Layout: layout, FullName: file_name, FileName: name, FileExt: ext, MaxSize: max_size}}
}

func NewFixSizeFileAppender(layout, file_name string, max_size int64) *FixSizeFileAppender {
	ret := new(FixSizeFileAppender)
	name, ext := split_file_name(file_name)
	ret.FileAppender = FileAppender{Layout: layout, FullName: file_name, FileName: name, FileExt: ext, MaxSize: max_size}
	ret.current_file_name = file_name
	return ret
}

func NewSplittedFileAppender(layout, file_name string, duration time.Duration) *SplittedFileAppender {
	name, ext := split_file_name(file_name)
	return &SplittedFileAppender{FileAppender: FileAppender{Layout: layout, FullName: file_name, FileName: name, FileExt: ext}, duration: duration, current_file_name: file_name}
}
