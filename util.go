package logger

import (
	"fmt"
	"strings"
	"time"

	config "github.com/i11cn/go_config"
)

func create_appender(cfg config.Config) (Appender, error) {
	typ := ""
	if err := cfg.GetAs(&typ, "type"); err != nil {
		return nil, fmt.Errorf("没有设置Appender的类型")
	}
	layout := ""
	if err := cfg.GetAs(&layout, "layout"); err != nil {
		layout = "%N %T %L : %M"
	}
	switch typ {
	case "console":
		return NewConsoleAppender(layout), nil
	case "stderr":
		return NewStderrAppender(layout), nil
	case "file":
		file := ""
		if err := cfg.GetAs(&file, "file"); err != nil {
			return nil, fmt.Errorf("必须设置日志文件名")
		}
		dur := ""
		if err := cfg.Get(&dur, "roll.duration"); err != nil {
			dur = "24h"
		}
		d, _ := time.ParseDuration(dur)
		return NewSplittedFileAppender(layout, file, d), nil
	default:
		return nil, fmt.Errorf("未知的Appender类型: %s", typ)
	}
	return nil, nil
}

func set_logger_level(log *Logger, cfg config.Config) error {
	level := 0
	if err := cfg.Get(&level, "level"); err != nil {
		levels := ""
		if err := cfg.Get(&levels, "level"); err != nil {
			return err
		}
		level_map := map[string]int{
			"ALL":   0,
			"TRACE": 10,
			"DEBUG": 20,
			"INFO":  30,
			"LOG":   40,
			"WARN":  50,
			"ERROR": 60,
			"FATAL": 70,
			"NONE":  100,
		}
		if l, exist := level_map[strings.ToUpper(levels)]; exist {
			log.SetLevel(l)
		} else {
			return fmt.Errorf("日志级别配置错误： %s", levels)
		}
	} else {
		log.SetLevel(level)
	}
	return nil
}

func NewLoggerFromConfig(cfg config.Config) (*Logger, error) {
	name := ""
	if err := cfg.GetAs(&name, "name"); err != nil {
		return nil, err
	}
	ret := GetLogger(name)
	if err := set_logger_level(ret, cfg); err != nil {
		return nil, err
	}
	if apds := cfg.SubArray("appenders"); apds != nil && len(apds) > 0 {
		for _, apd := range apds {
			if use, err := create_appender(apd); err != nil {
				return nil, err
			} else {
				ret.AddAppender(use)
			}
		}
	} else {
		fmt.Println(apds)
		return nil, fmt.Errorf("没有设置Appender，请至少设置一个Appender，如果目的是不想输出日志，请通过Level来控制")
	}
	return ret, nil
}

func NewLoggerFromYaml(in []byte) (*Logger, error) {
	if cfg, err := config.NewConfig().LoadYaml(in); err != nil {
		return nil, err
	} else {
		return NewLoggerFromConfig(cfg)
	}
}

func NewLoggerFromJson(in []byte) (*Logger, error) {
	if cfg, err := config.NewConfig().LoadJson(in); err != nil {
		return nil, err
	} else {
		return NewLoggerFromConfig(cfg)
	}
}

func NewLoggerFromYamlFile(file string) (*Logger, error) {
	if cfg, err := config.NewConfig().LoadYamlFile(file); err != nil {
		return nil, err
	} else {
		return NewLoggerFromConfig(cfg)
	}
}

func NewLoggerFromJsonFile(file string) (*Logger, error) {
	if cfg, err := config.NewConfig().LoadJsonFile(file); err != nil {
		return nil, err
	} else {
		return NewLoggerFromConfig(cfg)
	}
}
