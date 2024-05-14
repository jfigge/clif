/*
 * Copyright (C) 2024 by Jason Figge
 */

package clif

import (
	"fmt"
	"sync"

	"github.com/jfigge/clif/constants/color"
	"github.com/jfigge/clif/constants/screen"
)

var (
	ErrLoggerUnmarshalType = fmt.Errorf("logger error - invalid type")
)

type Logger struct {
	history *History
	sync    sync.Mutex
	debug   bool
}
type loggerConfigurationData struct {
	level     string `env:"${APPNAME}_LOGGER_LEVEL" file:"logger_level" cmd:"--logger-level" monitored:""`
	colorized bool   `env:"${APPNAME}_LOGGER_COLORIZED" file:"logger_colorized" cmd:"--logger-colorized"`
}
type LoggerConfiguration struct {
	*loggerConfigurationData
}

// ****** Construction ********************************************************

func NewLogger() (*Logger, error) {
	logger := &Logger{}

	return logger, nil
}

// ****** Log functions *******************************************************

func (l *Logger) Notify(text string, colour string) {
	str := fmt.Sprintf("%s%s%s%s%s", screen.ClearLine, colour, text, color.Reset, screen.ClearToEnd)
	l.history.add(str)
}
func (l *Logger) Tracef(text string, a ...interface{}) {
	l.Trace(fmt.Sprintf(text, a...))
}
func (l *Logger) Trace(text string) {
	if l.debug {
		l.history.add(fmt.Sprintf("%s%s%s%s", color.White, text, color.Reset, screen.ClearToEnd))
	}
}
func (l *Logger) Debugf(text string, a ...interface{}) {
	l.Debug(fmt.Sprintf(text, a...))
}
func (l *Logger) Debug(text string) {
	if l.debug {
		l.Notify(text, color.White)
	} else {
		l.history.add(fmt.Sprintf("%s%s%s%s", color.White, text, color.Reset, screen.ClearToEnd))
	}
}
func (l *Logger) Infof(text string, a ...interface{}) {
	l.Info(fmt.Sprintf(text, a...))
}
func (l *Logger) Info(text string) {
	l.Notify(text, color.BrightWhite)
}
func (l *Logger) Warnf(text string, a ...interface{}) {
	l.Warn(fmt.Sprintf(text, a...))
}
func (l *Logger) Warn(text string) {
	l.Notify(text, color.BrightYellow)
}
func (l *Logger) Errorf(text string, a ...interface{}) {
	l.Error(fmt.Sprintf(text, a...))
}
func (l *Logger) Error(text string) {
	l.Notify(text, color.BrightRed)
}

// ****** Configuration *******************************************************

func newLoggerConfiguration(values map[string]interface{}) (*LoggerConfiguration, error) {
	c := &LoggerConfiguration{loggerConfigurationData: &loggerConfigurationData{}}
	for key, value := range values {
		switch key {
		case "level":
			if level, err := toString(key, value); err != nil {
				return nil, fmt.Errorf("%w: %s", ErrLoggerUnmarshalType, err)
			} else {
				c.level = level
			}
		case "colorized":
			if colorized, err := toBool(key, value); err != nil {
				return nil, fmt.Errorf("%w: %s", ErrLoggerUnmarshalType, err)
			} else {
				c.colorized = colorized
			}
		}
	}
	return c, nil
}
func (c *LoggerConfiguration) Level() string {
	return c.level
}
func (c *LoggerConfiguration) SetLevel(level string) {
	c.level = level
}
func (c *LoggerConfiguration) Colorized() bool {
	return c.colorized
}

// ****** History *************************************************************

type History struct {
	sync     sync.Mutex
	messages []string
}

func (h *History) add(message string) {
	h.sync.Lock()
	defer h.sync.Unlock()

	h.messages = append(h.messages, message)
	if len(h.messages) > 1000 {
		h.messages = h.messages[1:]
	}
}
