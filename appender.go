package logger

import (
	"bytes"
	"fmt"
	"time"
)

type (
	Layout struct {
		Parts []func(int_args [2]int, str_args [5]string, args ...interface{}) []byte
	}

	Appender interface {
		Write(msg []byte)
		Format([2]int, [5]string, ...interface{}) []byte
		FormatBy(Layout, [2]int, [5]string, ...interface{}) []byte
		SetFormatter(Formatter)
	}

	BaseAppender struct {
		layout Layout
		format Formatter
	}
)

func NewBaseAppender(layout string) BaseAppender {
	return BaseAppender{Layout{ParseLayout(layout)}, NewStringFormatter()}
}

func (ba *BaseAppender) Format(int_args [2]int, str_args [5]string, args ...interface{}) []byte {
	return ba.format.Format(ba.layout, int_args, str_args, args...)
}

func (ba *BaseAppender) FormatBy(layout Layout, int_args [2]int, str_args [5]string, args ...interface{}) []byte {
	return ba.format.Format(layout, int_args, str_args, args...)
}

func (ba *BaseAppender) SetFormatter(format Formatter) {
	ba.format = format
}

func ParseLayout(l string) []func([2]int, [5]string, ...interface{}) []byte {
	ret := make([]func([2]int, [5]string, ...interface{}) []byte, 0, 10)
	tag := false
	buf := new(bytes.Buffer)
	current := 0
	add := func(fn func([2]int, [5]string, ...interface{}) []byte) {
		if buf.Len() > 0 {
			msg := buf.Bytes()
			ret = append(ret, func([2]int, [5]string, ...interface{}) []byte {
				return msg
			})
			buf = new(bytes.Buffer)
		}
		ret = append(ret, fn)

	}
	for _, c := range l {
		if tag {
			// 处理%*中的那个*
			switch c {
			case 'F':
				add(func(a1 [2]int, str_args [5]string, a3 ...interface{}) []byte {
					return []byte(str_args[0])
				})

			case 'f':
				add(func(a1 [2]int, str_args [5]string, a3 ...interface{}) []byte {
					return []byte(str_args[1])
				})

			case 'l':
				add(func(int_args [2]int, a2 [5]string, a3 ...interface{}) []byte {
					return []byte(fmt.Sprint(int_args[0]))
				})

			case 'N':
				add(func(a1 [2]int, str_args [5]string, a3 ...interface{}) []byte {
					return []byte(str_args[2])
				})

			case 'L':
				add(func(a1 [2]int, str_args [5]string, a3 ...interface{}) []byte {
					return []byte(str_args[3])
				})

			case 'p':
				add(func(int_args [2]int, a2 [5]string, a3 ...interface{}) []byte {
					return []byte(fmt.Sprint(int_args[1]))
				})

			case 'T':
				add(func(a1 [2]int, str_args [5]string, a3 ...interface{}) []byte {
					return []byte(str_args[4])
				})

			case '%':
				add(func(a1 [2]int, str_args [5]string, a3 ...interface{}) []byte {
					return []byte("%")
				})

			case 'm':
				add(func(pos int) func([2]int, [5]string, ...interface{}) []byte {
					return func(a1 [2]int, a2 [5]string, args ...interface{}) []byte {
						if pos < len(args) {
							return []byte(fmt.Sprint(args[pos]))
						} else {
							return []byte("%m")
						}
					}
				}(current))
				current++

			case 'M':
				add(func(pos int) func([2]int, [5]string, ...interface{}) []byte {
					return func(a1 [2]int, a2 [5]string, args ...interface{}) []byte {
						if pos < len(args) {
							return []byte(fmt.Sprint(args[pos:]...))
						} else {
							return []byte("%M")
						}
					}
				}(current))
				current = 0x7FFF

			default:
				buf.WriteRune(c)
			}
			tag = false
		} else if c == '%' {
			// 遇到了%*中的那个%
			tag = true
		} else {
			// 啥都没有，正常的一个字符，照原样输出
			buf.WriteRune(c)
		}
	}
	if buf.Len() > 0 {
		msg := buf.Bytes()
		ret = append(ret, func([2]int, [5]string, ...interface{}) []byte {
			return msg
		})
	}
	return ret
}

func NewConsoleAppender(layout string) *ConsoleAppender {
	return &ConsoleAppender{NewBaseAppender(layout)}
}

func NewFileAppender(layout, file_name string, max_size int64) *FileAppender {
	name, ext := split_file_name(file_name)
	return &FileAppender{BaseAppender: NewBaseAppender(layout), FullName: file_name, FileName: name, FileExt: ext, MaxSize: max_size}
}

func NewTruncatedFileAppender(layout, file_name string, max_size int64) *TruncatedFileAppender {
	return &TruncatedFileAppender{*(NewFileAppender(layout, file_name, max_size))}
}

func NewFixSizeFileAppender(layout, file_name string, max_size int64) *FixSizeFileAppender {
	ret := &FixSizeFileAppender{FileAppender: *(NewFileAppender(layout, file_name, max_size))}
	ret.current_file_name = file_name
	return ret
}

func NewSplittedFileAppender(layout, file_name string, duration time.Duration) *SplittedFileAppender {
	return &SplittedFileAppender{FileAppender: *(NewFileAppender(layout, file_name, 0)), duration: duration, current_file_name: file_name}
}
