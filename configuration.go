/*
 * Copyright (C) 2024 by Jason Figge
 */

package clif

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"time"
	"unsafe"

	"github.com/fsnotify/fsnotify"
	"gopkg.in/yaml.v3"
)

var (
	environmentInterval  = 5 * time.Second
	configurationType    = reflect.TypeOf(Configuration{})
	configurationTypePtr = reflect.TypeOf(&Configuration{})
)

type processFunc func(value reflect.Value, field reflect.StructField, level int, key string) error
type ConfigurationOption func(c *Configuration) error
type ConfigurationNotifyFunc func(setting string, value interface{})
type configurationData struct {
	lock        sync.Mutex
	notifyFuncs map[string][]ConfigurationNotifyFunc
}
type configurationMetadata struct {
	appName    string
	configFile string
	homeDir    string
	configDir  string
	load       bool
	watch      bool
	wg         *sync.WaitGroup
	watcher    *fsnotify.Watcher
}
type Configuration struct {
	*configurationData
	Metadata *configurationMetadata
	Logger   *LoggerConfiguration  `json:"logger" yaml:"logger"`
	Console  *ConsoleConfiguration `json:"console" yaml:"console"`
}

// ****** Construction ********************************************************

func InitConfig(ctx context.Context, configuration interface{}, options ...ConfigurationOption) error {
	rv := reflect.ValueOf(configuration)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return &InvalidInitConfigError{Type: reflect.TypeOf(configuration)}
	}

	var c *Configuration
	var found bool
	for i := 0; i < rv.Elem().NumField(); i++ {
		field := rv.Elem().Field(i)
		if field.CanConvert(configurationTypePtr) {
			ft := reflect.TypeOf(configuration).Elem().Field(i)
			if ft.Anonymous {
				return &InvalidInitConfigError{Code: ErrAnonymousCoreConfig}
			}
			if c == nil {
				c = &Configuration{}
				if !ft.IsExported() {
					return &InvalidInitConfigError{Code: ErrNonExportedCoreConfig}
					// field = reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem()
				}
				field.Set(reflect.ValueOf(c))
			}
			if err := c.setMetadataDefaults(); err != nil {
				return err
			}
			if err := c.newConfiguration(ctx, options...); err != nil {
				return err
			}
			if c.Metadata.load {
				var cfg interface{}
				cfg, err := c.unmarshalConfigFile(ctx, configuration)
				if err != nil {
					return err
				}
				fmt.Printf("%v\n", cfg)
			}
			found = true
			break

		} else if field.CanConvert(configurationType) {
			return &InvalidInitConfigError{Type: configurationType}
		}
	}

	if !found {
		return &InvalidInitConfigError{Code: ErrMissingCoreConfig}
	}

	if c.Metadata.load {
		if err := walkStructure(configuration, 0, "", processDefault, processEnvVar); err != nil {
			return err
		}
	}
	return nil
}

func walkStructure(s interface{}, level int, key string, fs ...processFunc) error {
	rv := reflect.ValueOf(s)
	rt := reflect.TypeOf(s)
	if rv.Kind() == reflect.Pointer {
		rv = rv.Elem()
		rt = rt.Elem()
	}
	var err error
	for i := 0; err == nil && i < rv.NumField(); i++ {
		fv := rv.Field(i)
		ft := rt.Field(i)
		if fv.CanConvert(configurationTypePtr) && !ft.IsExported() {
			fv = reflect.NewAt(fv.Type(), unsafe.Pointer(fv.UnsafeAddr())).Elem()
		} else if !ft.IsExported() {
			continue
		}
		switch fv.Kind() {
		case reflect.Struct:
			fvp := reflect.New(fv.Type())
			fvp.Elem().Set(fv)
			err = walkStructure(fvp.Interface(), level+1, key+".", fs...)
			fv.Set(fvp.Elem())
		case reflect.Pointer:
			if fv.CanSet() {
				if fv.IsNil() {
					newInstance := reflect.New(ft.Type.Elem())
					fv.Set(newInstance)
				}
				err = walkStructure(fv.Interface(), level+1, key+".", fs...)
			}
		default:
			for _, f := range fs {
				err = f(fv, ft, level, key)
				if err != nil {
					break
				}
			}
		}
	}
	return err
}

func processDefault(fv reflect.Value, ft reflect.StructField, level int, key string) error {
	return nil
}

func processEnvVar(fv reflect.Value, ft reflect.StructField, level int, key string) error {
	return nil
}

func (c *Configuration) setMetadataDefaults() error {
	currentUser, err := user.Current()
	if err != nil {
		return err
	}
	if c.Metadata == nil {
		c.Metadata = &configurationMetadata{}
	}
	c.Metadata.load = true
	c.Metadata.watch = true
	c.Metadata.appName = os.Args[0]
	c.Metadata.configFile = "config.yaml"
	c.Metadata.homeDir = currentUser.HomeDir
	c.Metadata.configDir = filepath.Join(c.Metadata.homeDir, c.Metadata.configFile)
	return nil
}

func (c *Configuration) newConfiguration(ctx context.Context, options ...ConfigurationOption) error {
	if c.configurationData == nil {
		c.configurationData = &configurationData{
			notifyFuncs: make(map[string][]ConfigurationNotifyFunc),
		}
	}

	for _, option := range options {
		if err := option(c); err != nil {
			return err
		}
	}

	if c.Metadata.watch {
		var err error
		c.Metadata.watcher, err = fsnotify.NewWatcher()
		if err != nil {
			return err
		}
		go c.watch(ctx)
	}

	return nil
}
func (c *Configuration) watch(ctx context.Context) {
	if c.Metadata.wg != nil {
		c.Metadata.wg.Add(1)
		defer c.Metadata.wg.Done()
	}
	timer := time.NewTicker(environmentInterval)
	for {
		select {
		case event, ok := <-c.Metadata.watcher.Events:
			if !ok {
				return
			}
			fmt.Println("event:", event)
			if event.Has(fsnotify.Write) {
				log.Println("modified file:", event.Name)
			}
		case err, ok := <-c.Metadata.watcher.Errors:
			if !ok {
				return
			}
			fmt.Println("error:", err)
		case <-timer.C:
			timer.Stop()
			c.checkForEnvChange()
			timer.Reset(environmentInterval)
		case <-ctx.Done():
			_ = c.Metadata.watcher.Close()
			return
		}
	}
}
func (c *Configuration) checkForEnvChange() {
	for setting, notifyFuncs := range c.configurationData.notifyFuncs {
		value := os.Getenv(setting)
		for _, notifyFunc := range notifyFuncs {
			notifyFunc(setting, value)
		}
	}
}
func (c *Configuration) configType() string {
	index := strings.LastIndex(c.Metadata.configFile, ".")
	if index == -1 {
		return ""
	}
	return strings.ToLower(c.Metadata.configFile[index+1:])
}

// ****** Metadata functions **************************************************

func (m *configurationMetadata) AppName() string {
	return m.appName
}
func (m *configurationMetadata) HomeDir() string {
	return m.homeDir
}
func (m *configurationMetadata) ConfigDir() string {
	return m.configDir
}
func (m *configurationMetadata) ConfigFile() string {
	return m.configFile
}

// ****** Configuration unmarshal functions ***********************************

func (c *Configuration) unmarshalConfigFile(ctx context.Context, config interface{}) (interface{}, error) {
	if c.Metadata.configFile != "" {
		bs, err := os.ReadFile(c.Metadata.configFile)
		if err != nil {
			return nil, &InvalidInitConfigError{Code: ErrFileReadCoreConfig, err: err}
		}
		switch c.configType() {
		case "json":
			err = json.Unmarshal(bs, config)
		case "yaml", "yml":
			err = yaml.Unmarshal(bs, config)
		}
		if err != nil {
			return nil, &InvalidInitConfigError{Code: ErrUnmarshalCoreConfig, err: err}
		}
	}
	return config, nil
}
func (c *Configuration) UnmarshalJSON(bs []byte) error {
	data := map[string]interface{}{}
	err := json.Unmarshal(bs, &data)
	if err != nil {
		return err
	}
	return c.unmarshalLoad(data)
}
func (c *Configuration) UnmarshalYAML(value *yaml.Node) error {
	data := map[string]interface{}{}
	err := value.Decode(&data)
	if err != nil {
		return err
	}
	return c.unmarshalLoad(data)
}
func (c *Configuration) unmarshalLoad(values map[string]interface{}) error {
	var err error
	for key, value := range values {
		switch key {
		case "logger":
			if m, ok := value.(map[string]interface{}); ok {
				c.Logger, err = newLoggerConfiguration(m)
			} else {
				return InvalidInitConfigError{
					Code: ErrUnmarshalLoggerData,
					err:  fmt.Errorf("%s != %s", key, "map[string]interface{}"),
				}
			}
		case "console":
			if m, ok := value.(map[string]interface{}); ok {
				c.Console, err = newConsoleConfiguration(m)
			} else {
				return InvalidInitConfigError{
					Code: ErrUnmarshalLoggerData,
					err:  fmt.Errorf("%s != %s", key, "map[string]interface{}"),
				}
			}
		}
	}
	return err
}

// AddNotifyOnChange add a monitor to the named setting and triggers the notifyFunc when the value changes
func (c *configurationData) AddNotifyOnChange(setting string, notifyFunc ConfigurationNotifyFunc) {
	c.lock.Lock()
	defer c.lock.Unlock()

	notifyFuncs, ok := c.notifyFuncs[setting]
	if !ok {
		notifyFuncs = make([]ConfigurationNotifyFunc, 0)
	}
	c.notifyFuncs[setting] = append(notifyFuncs, notifyFunc)
}

// ****** Options *************************************************************

func ConfigurationOptionWaitGroup(wg *sync.WaitGroup) ConfigurationOption {
	return func(c *Configuration) error {
		c.Metadata.wg = wg
		return nil
	}
}
func ConfigurationOptionAppName(appName string) ConfigurationOption {
	return func(c *Configuration) error {
		c.Metadata.appName = appName
		return nil
	}
}
func ConfigurationOptionConfigDir(configDir string) ConfigurationOption {
	return func(c *Configuration) error {
		c.Metadata.configDir = configDir
		return nil
	}
}
func ConfigurationOptionConfigFile(configFile string) ConfigurationOption {
	return func(c *Configuration) error {
		c.Metadata.configFile = configFile
		return nil
	}
}
func configurationOptionNoLoad() ConfigurationOption {
	return func(c *Configuration) error {
		c.Metadata.load = false
		return nil
	}
}
func configurationOptionNoWatch() ConfigurationOption {
	return func(c *Configuration) error {
		c.Metadata.watch = false
		return nil
	}
}

// ***** Error ****************************************************************

const (
	ErrMissingCoreConfig     = "CC01"
	ErrAnonymousCoreConfig   = "CC02"
	ErrUnmarshalCoreConfig   = "CC03"
	ErrUnmarshalLoggerData   = "CC03LC01"
	ErrUnmarshalConsoleData  = "CC03CC01"
	ErrFileReadCoreConfig    = "CC04"
	ErrNonExportedCoreConfig = "CC05"
)

type InvalidInitConfigError struct {
	Code string
	err  error
	Type reflect.Type
}

func (e InvalidInitConfigError) Error() string {
	switch e.Code {
	case ErrMissingCoreConfig:
		return "configuration error - InitConfig(core configuration not present)"
	case ErrAnonymousCoreConfig:
		return "configuration error - InitConfig(core configuration is anonymous)"
	case ErrNonExportedCoreConfig:
		return "configuration error - InitConfig(core configuration is unexported)"
	case ErrUnmarshalCoreConfig, ErrUnmarshalLoggerData, ErrUnmarshalConsoleData,
		ErrFileReadCoreConfig:
		return fmt.Sprintf("configuration error - %v", e.err)
	default:
		if e.Type == nil {
			return "configuration error - InitConfig(nil)"
		}

		if e.Type == configurationType {
			return "configuration error - InitConfig(non-pointer of embedded " + e.Type.String() + ")"
		} else if e.Type.Kind() != reflect.Pointer {
			return "configuration error - InitConfig(non-pointer " + e.Type.String() + ")"
		}

		return "configuration error - InitConfig(nil " + e.Type.String() + ")"
	}
}
