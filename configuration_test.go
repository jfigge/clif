/*
 * Copyright (C) 2024 by Jason Figge
 */

package clif

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

type configA struct {
}

type configB struct {
	name string
}

type configC struct {
	Configuration
	name string
}

type configCPtr struct {
	*Configuration
	name string
}

type configD struct {
	core Configuration
	name string
}

type configDPtr struct {
	name string
	core *Configuration
}

type configEPtr struct {
	name string
	Core *Configuration
}

func Test_InitConfig(t *testing.T) {
	tests := map[string]struct {
		config   interface{}
		validate func(t *testing.T, c interface{})
		errStr   string
	}{
		"nil": {
			config: nil,
			errStr: "configuration error - InitConfig(nil)",
		},
		"no fields": {
			config: configA{},
			errStr: "configuration error - InitConfig(non-pointer clif.configA)",
		},
		"non-pointer": {
			config: configB{},
			errStr: "configuration error - InitConfig(non-pointer clif.configB)",
		},
		"no core": {
			config: &configB{},
			errStr: "configuration error - InitConfig(core configuration not present)",
		},
		"non-pointer embedded": {
			config: &configC{},
			errStr: "configuration error - InitConfig(non-pointer of embedded clif.Configuration)",
		},
		"pointer embedded": {
			config: &configCPtr{},
			errStr: "configuration error - InitConfig(core configuration is anonymous)",
		},
		"non-pointer field": {
			config: &configD{},
			errStr: "configuration error - InitConfig(non-pointer of embedded clif.Configuration)",
		},
		"Internal pointer field": {
			config: &configDPtr{},
			errStr: "",
			validate: func(t *testing.T, test interface{}) {
				if c, ok := test.(*configDPtr); !ok {
					t.Fatal("invalid test")
				} else {
					assert.NotNil(t, c)
					assert.NotNil(t, c.core)
					assert.NotNil(t, c.core.Metadata)
					assert.NotNil(t, c.core.Metadata.configFile)
				}
			},
		},
		"External pointer field": {
			config: &configEPtr{},
			errStr: "",
			validate: func(t *testing.T, test interface{}) {
				if c, ok := test.(*configEPtr); !ok {
					t.Fatal("invalid test")
				} else {
					assert.NotNil(t, c)
					assert.NotNil(t, c.Core)
					assert.NotNil(t, c.Core.Metadata)
					assert.NotNil(t, c.Core.Metadata.configFile)
				}
			},
		},
	}
	for name, test := range tests {
		t.Run(name, func(tt *testing.T) {
			ctx := context.Background()
			err := InitConfig(
				ctx, test.config,
				configurationOptionSkipLoad(),
				configurationOptionNoWatch())
			if test.errStr != "" {
				if err != nil {
					assert.Equal(tt, test.errStr, err.Error())
				} else {
					assert.Error(tt, err)
				}
			} else {
				assert.NoError(tt, err)
				if test.validate != nil {
					test.validate(t, test.config)
				}
			}
		})
	}
}

type WalkerC struct {
	internal string
	External string
}

type WalkerB struct {
	internal   string
	External   string
	WalkerC    WalkerC
	WalkerCPtr *WalkerC
}

type WalkerA struct {
	internal   string
	External   string
	Mappy      map[string]interface{}
	Things     []string
	Object     interface{}
	handler    func() error
	Number     int
	Boolean    bool
	WalkerB    WalkerB
	WalkerBPtr *WalkerB
	core       *Configuration
}

func Test_Walker(t *testing.T) {
	ctx := context.Background()
	tests := map[string]struct {
		config interface{}
	}{
		"success": {&WalkerA{WalkerBPtr: &WalkerB{}}},
	}
	for name, test := range tests {
		t.Run(name, func(tt *testing.T) {
			err := InitConfig(
				ctx, test.config,
				configurationOptionNoWatch(),
			)
			fmt.Printf("Error: %v\n", err)
		})
	}
}

func f(w *WalkerA) {
	w.WalkerB.WalkerC.External = "2.1"
	w.WalkerB.WalkerCPtr = &WalkerC{External: "2."}
	w.WalkerBPtr.WalkerC.External = "2.1"
	w.WalkerBPtr.WalkerCPtr = &WalkerC{External: "2."}
}
