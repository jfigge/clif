/*
 * Copyright (C) 2024 by Jason Figge
 */

package clif

import (
	"context"
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
	Core Configuration
	name string
}

type configDPtr struct {
	name string
	Core *Configuration
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
					assert.NotNil(t, c.Core)
					assert.NotNil(t, c.Core.Metadata)
					assert.NotNil(t, c.Core.Metadata.configFile)
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
				configurationOptionNoLoad(),
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
	External   string   `yaml:"external" json:"external"`
	WalkerC    WalkerC  `yaml:"walkerC" json:"walkerC"`
	WalkerCPtr *WalkerC `yaml:"walkerCPtr" json:"walkerCPtr"`
}

type WalkerA struct {
	internal   string
	External   string                 `yaml:"external" json:"external"`
	Mappy      map[string]interface{} `yaml:"mappy" json:"mappy"`
	Things     []string               `yaml:"things" json:"things"`
	Object     interface{}            `yaml:"object" json:"object"`
	handler    func() error
	Number     int            `yaml:"number" json:"number"`
	Boolean    bool           `yaml:"boolean" json:"boolean"`
	WalkerB    WalkerB        `yaml:"walkerB" json:"walkerB"`
	WalkerBPtr *WalkerB       `yaml:"walkerBPtr" json:"walkerBPtr"`
	Core       *Configuration `yaml:"core" json:"core"`
}

func Test_Walker(t *testing.T) {
	ctx := context.Background()
	tests := map[string]struct {
		config   interface{}
		filename string
	}{
		"yaml": {
			config:   &WalkerA{WalkerBPtr: &WalkerB{}},
			filename: "testdata/config.yaml",
		},
		//"json": {
		//	config:   &WalkerA{WalkerBPtr: &WalkerB{}},
		//	filename: "testdata/config.json",
		//},
	}
	for name, test := range tests {
		t.Run(name, func(tt *testing.T) {
			err := InitConfig(
				ctx, test.config,
				configurationOptionNoWatch(),
				ConfigurationOptionConfigFile(test.filename),
			)
			actual, ok := test.config.(*WalkerA)
			if !ok {
				t.Fatal("unable to convert config type")
			} else if err != nil {
				t.Fatalf("%v", err)
			}
			assert.Equal(tt, "walkerA.external", actual.External)
			assert.Equal(tt, map[string]interface{}{"int": 1, "bool": true, "string": "string"}, actual.Mappy)
			assert.Equal(tt, []string{"one", "two", "three"}, actual.Things)
			assert.Equal(tt, "blah", actual.Object)
			assert.Equal(tt, -1, actual.Number)
			assert.Equal(tt, true, actual.Boolean)
			assert.Equal(tt, "walkerB.external", actual.WalkerB.External)
			assert.Equal(tt, "walkerBC.external", actual.WalkerB.WalkerC.External)
			assert.Equal(tt, "walkerBCPtr.external", actual.WalkerB.WalkerCPtr.External)
			assert.Equal(tt, "walkerBPtr.external", actual.WalkerBPtr.External)
			assert.Equal(tt, "walkerBPtrC.external", actual.WalkerBPtr.WalkerC.External)
			assert.Equal(tt, "walkerBPtrCPtr.external", actual.WalkerBPtr.WalkerCPtr.External)
			assert.Equal(tt, "debug", actual.Core.Logger.level)
			assert.Equal(tt, true, actual.Core.Logger.colorized)
			assert.Equal(tt, 50, actual.Core.Console.width)
			assert.Equal(tt, 10, actual.Core.Console.height)
		})
	}
}
