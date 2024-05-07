/*
 * Copyright (C) 2024 by Jason Figge
 */

package clif

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/jfigge/clif/constants/color"
	"github.com/jfigge/clif/constants/cursor"
	"github.com/jfigge/clif/constants/screen"
	"golang.org/x/term"
)

var (
	singletonConsole *Console
	synchronize      sync.Mutex
)

var (
	ErrConsoleNoTerminal       = fmt.Errorf("console error - no terminal available")
	ErrConsoleInvalidRawMode   = fmt.Errorf("console error - unable to enter raw mode")
	ErrConsoleSizeUnavailable  = fmt.Errorf("console error - unable to obtain dimensions")
	ErrConsoleSizeNonCompliant = fmt.Errorf("console warn - requested size not met")
)

type ConsoleOption func(c *Console) error
type ConsoleResizeFunc func(width, height int)
type ConsoleStopFunc func()
type ConsoleWaitGroup func(wg *sync.WaitGroup)
type consoleConfigurationData struct {
	width  int `json:"width" yaml:"width" cmd:"--console-width"`
	height int `json:"height" yaml:"height" cmd:"--console-height"`
}
type ConsoleConfiguration struct {
	*consoleConfigurationData
}
type Console struct {
	fd         int
	oldState   *term.State
	monitor    bool
	width      int
	height     int
	lastWidth  int
	lastHeight int
	resizeFunc ConsoleResizeFunc
	stopFunc   ConsoleStopFunc
	stopCancel context.CancelFunc
	wg         *sync.WaitGroup
}

// ****** Construction ********************************************************

func NewConsole(ctx context.Context, options ...ConsoleOption) (*Console, error) {
	synchronize.Lock()
	defer synchronize.Unlock()

	if singletonConsole == nil {
		c := &Console{}
		for _, option := range options {
			err := option(c)
			if err != nil {
				return nil, err
			}
		}
		err := c.startConsole(ctx)
		if err != nil {
			return nil, err
		}
		go c.autoStop(ctx)

		err = c.waitForSize()
		if err != nil {
			c.stopConsole()
			return nil, err
		}

		if c.resizeFunc != nil {
			var stopCtx context.Context
			stopCtx, c.stopCancel = context.WithCancel(ctx)
			go c.monitorSize(stopCtx)
		}
		singletonConsole = c

	}

	return singletonConsole, nil
}
func (c *Console) startConsole(ctx context.Context) error {
	c.fd = int(os.Stdin.Fd())
	if !term.IsTerminal(c.fd) {
		return ErrConsoleNoTerminal
	}

	var err error
	c.oldState, err = term.MakeRaw(c.fd)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrConsoleInvalidRawMode, err)
	}
	return nil
}
func (c *Console) autoStop(ctx context.Context) {
	if c.wg != nil {
		c.wg.Add(1)
		defer c.wg.Done()
	}
	select {
	case <-ctx.Done():
		c.StopConsole()
		if c.stopCancel != nil {
			c.stopCancel()
		}
	}
}
func (c *Console) monitorSize(ctx context.Context) {
	if c.wg != nil {
		c.wg.Add(1)
		defer c.wg.Done()
	}
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	var width int
	var height int
	var err error
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if width, height, err = term.GetSize(c.fd); err != nil {
				continue
			}
			if c.lastWidth != width || c.lastHeight != height {
				c.lastWidth = width
				c.lastHeight = height
				if c.resizeFunc != nil {
					c.resizeFunc(width, height)
				}
			}
		}
	}
}
func (c *Console) waitForSize() error {
	if c.wg != nil {
		c.wg.Add(1)
		defer c.wg.Done()
	}
	var err error
	timeout := time.Now().Add(30 * time.Second)
	for timeout.After(time.Now()) {
		c.lastWidth, c.lastHeight, err = term.GetSize(c.fd)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrConsoleSizeUnavailable, err)
		}
		validWidth, validHeigh := color.Green, color.Green
		if c.lastWidth < c.width {
			validWidth = color.Red
		}
		if c.lastHeight < c.height {
			validHeigh = color.Red
		}
		c.PrintAtf(0, 0, "%s%s%d%sx%s%d%s",
			screen.ClearScreen, validWidth, c.lastWidth, color.Reset, validHeigh, c.lastHeight, color.Reset,
		)
		if c.lastWidth >= c.width && c.lastHeight >= c.height {
			return nil
		}
		time.Sleep(time.Millisecond * 50)
	}
	return ErrConsoleSizeUnavailable
}
func (c *Console) StopConsole() {
	synchronize.Lock()
	defer synchronize.Unlock()
	c.stopConsole()
}
func (c *Console) stopConsole() {
	c.PrintAtf(0, 0, cursor.Show)
	if c.stopFunc != nil {
		c.stopFunc()
	}
	_ = term.Restore(c.fd, c.oldState)
	singletonConsole = nil
}

// ****** Console functions ***************************************************

func (c *Console) PrintAtf(x int, y int, format string, args ...interface{}) {
	hdr := fmt.Sprintf(screen.At, y+1, x+1, format)
	fmt.Printf(hdr, args...)

}

// ****** Configuration *******************************************************

func (c *ConsoleConfiguration) Width() int {
	return c.width
}
func (c *ConsoleConfiguration) SetWidth(width int) {
	c.width = width
}
func (c *ConsoleConfiguration) Height() int {
	return c.height
}
func (c *ConsoleConfiguration) SetHeight(height int) {
	c.height = height
}

// ****** Options *************************************************************

func ConsoleOptionSize(width, height int) ConsoleOption {
	return func(c *Console) error {
		if c.width != width || c.height != height {
			c.monitor = true
		}
		c.width = width
		c.height = height
		return nil
	}
}
func ConsoleOptionResizeNotify(resizeFunc ConsoleResizeFunc) ConsoleOption {
	return func(c *Console) error {
		c.resizeFunc = resizeFunc
		return nil
	}
}
func ConsoleOptionStopNotify(stopFunc ConsoleStopFunc) ConsoleOption {
	return func(c *Console) error {
		c.stopFunc = stopFunc
		return nil
	}
}
func ConsoleOptionWaitGroup(wg *sync.WaitGroup) ConsoleOption {
	return func(c *Console) error {
		c.wg = wg
		return nil
	}
}
