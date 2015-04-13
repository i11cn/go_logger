package go_logger

import (
    "testing"
)

func Test_ConsoleAppender(t *testing.T) {
    l := GetLogger("test")
    l.ClearAppender()
    l.AddAppender(&ConsoleAppender{"%N %T %M"})
    l.Log("test ", "sample ", "log")
}

func Test_FileAppender(t *testing.T) {
    l := GetLogger("test")
    l.ClearAppender()
    l.AddAppender(&FileAppender{Layout: "%N %T %M", FileName: "test.log", MaxSize:128})
    l.Log("test ", "sample ", "log")
}

