/*
 * Copyright (C) 2024 by Jason Figge
 */

package clif

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/jfigge/clif/constants/keys"
)

type KeyboardOption func(k *Keyboard) error
type KeyboardStopFunc func()

type Keyboard struct {
	wg       *sync.WaitGroup
	kbChan   chan keys.Key
	reqChan  chan bool
	stopFunc KeyboardStopFunc
}

// ****** Construction ********************************************************

func NewKeyboard(ctx context.Context, options ...KeyboardOption) (*Keyboard, error) {
	k := &Keyboard{
		kbChan:  make(chan keys.Key),
		reqChan: make(chan bool),
	}

	for _, option := range options {
		err := option(k)
		if err != nil {
			return nil, err
		}
	}
	go k.readMonitoredInput(ctx)

	return k, nil
}
func (k *Keyboard) KeyboardChannel() <-chan keys.Key {
	return k.kbChan
}
func (k *Keyboard) RequestChannel() chan<- bool {
	return k.reqChan
}
func (k *Keyboard) readMonitoredInput(ctx context.Context) {
	if k.wg != nil {
		k.wg.Add(1)
		defer k.wg.Done()
	}
	var moreInput bool
	for {
		select {
		case <-ctx.Done():
			k.stopNotify()
			return
		case moreInput = <-k.reqChan:
			if !moreInput {
				k.stopNotify()
				return
			}
			readInput(k.kbChan)
		}
	}
}
func (k *Keyboard) stopNotify() {
	if k.stopFunc != nil {
		k.stopFunc()
	}
}
func readInput(kbChan chan<- keys.Key) {
	bs := make([]byte, 5)
	numRead, err := os.Stdin.Read(bs)
	if err != nil {
		fmt.Printf("unexpected failure to read input. Interactive mode not available\n")
		return
	}
	if key, ok := keys.MakeKey(bs[:numRead]); ok {
		kbChan <- *key
	}
}

// ****** Options *************************************************************

func KeyboardOptionWaitGroup(wg *sync.WaitGroup) KeyboardOption {
	return func(k *Keyboard) error {
		k.wg = wg
		return nil
	}
}
func KeyboardOptionStopNotify(stopFunc KeyboardStopFunc) KeyboardOption {
	return func(k *Keyboard) error {
		k.stopFunc = stopFunc
		return nil
	}
}
