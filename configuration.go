/*
 * Copyright (C) 2024 by Jason Figge
 */

package clif

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"reflect"
	"sync"
	"time"
	"unsafe"

	"github.com/fsnotify/fsnotify"
)

var (
	environmentInterval  = 5 * time.Second
	configurationType    = reflect.TypeOf(Configuration{})
	configurationTypePtr = reflect.TypeOf(&Configuration{})
)

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
	Logger   LoggerConfiguration  `json:"logger" yaml:"logger"`
	Console  ConsoleConfiguration `json:"console" yaml:"console"`
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
				return &InvalidInitConfigError{Code: errAnonymousCoreConfig}
			}
			if c == nil {
				c = &Configuration{}
				if !ft.IsExported() {
					field = reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem()
				}
				field.Set(reflect.ValueOf(c))
			}
			if err := c.setMetadataDefaults(); err != nil {
				return err
			}
			if err := c.loadConfiguration(ctx, options...); err != nil {
				return err
			}
			found = true
			break
		} else if field.CanConvert(configurationType) {
			return &InvalidInitConfigError{Type: configurationType}
		}
	}

	if !found {
		return &InvalidInitConfigError{Code: errMissingCoreConfig}
	}

	if c.Metadata.load {
		if err := walkStructure(configuration, loadEnvironmentVariables, 0, "", ""); err != nil {
			return err
		}
	}
	return nil
}

func walkStructure(s interface{}, providerFunc interface{}, level int, indent string, key string) error {
	rv := reflect.ValueOf(s)
	rt := reflect.TypeOf(s)
	if rv.Kind() == reflect.Pointer {
		rv = rv.Elem()
		rt = rt.Elem()
	}
	for i := 0; i < rv.NumField(); i++ {
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
			walkStructure(fvp.Interface(), providerFunc, level+1, indent+"   ", key+".")
			fv.Set(fvp.Elem())
		case reflect.Pointer:
			if fv.CanSet() {
				if fv.IsNil() {
					newInstance := reflect.New(ft.Type.Elem())
					fv.Set(newInstance)
				}
				walkStructure(fv.Interface(), providerFunc, level+1, indent+"   ", key+".")
			} else {
				fmt.Printf("???\n")
			}
		default:
			setFieldValue(fv, ft, providerFunc, level, indent, key)
		}
	}
	return nil
}

func setFieldValue(fv reflect.Value, ft reflect.StructField, providerFunc interface{}, level int, indent string, key string) {

}

func walkStructurX(s interface{}, providerFunc interface{}, level int, indent string, key string) error {
	rv := reflect.ValueOf(s)
	rt := reflect.TypeOf(s)
	fmt.Printf("%s%s %s\n", indent, rt.Name(), rv.Kind().String())
	if rv.Kind() == reflect.Pointer {
		rv = rv.Elem()
		rt = rt.Elem()
	}
	for i := 0; i < rv.NumField(); i++ {
		fv := rv.Field(i)
		ft := rt.Field(i)
		fmt.Printf("%s - Field: ", indent)
		if ft.IsExported() {
			fmt.Printf("+")
		} else if fv.CanConvert(configurationTypePtr) {
			fmt.Printf("+")
			fv = reflect.NewAt(fv.Type(), unsafe.Pointer(fv.UnsafeAddr())).Elem()
		} else {
			fmt.Printf("%s - Skipping\n", ft.Name)
			continue
		}
		if fv.CanSet() {
			fmt.Printf("!")
		}
		fmt.Printf("%s %s", ft.Name, fv.String())
		switch fv.Kind() {
		case reflect.Struct:
			fvp := reflect.New(fv.Type())
			fvp.Elem().Set(fv)
			walkStructure(fvp.Interface(), providerFunc, level+1, indent+"   ", key+".")
			fv.Set(fvp.Elem())
		case reflect.Pointer:
			if fv.CanSet() {
				if fv.IsNil() {
					newInstance := reflect.New(ft.Type.Elem())
					fv.Set(newInstance)
				}
				walkStructure(fv.Interface(), providerFunc, level+1, indent+"   ", key+".")
			} else {
				fmt.Printf("???\n")
			}
		case reflect.String:
			fv.Set(reflect.ValueOf(fmt.Sprintf("%d.%d", level, i)))
		default:
			fmt.Printf("%v\n", fv.Interface())
		}
	}
	return nil
}

func loadEnvironmentVariables(configuration interface{}) error {
	//	rv := reflect.ValueOf(configuration)
	//	if rv.Kind() != reflect.Pointer || rv.IsNil() {
	//		return &InvalidInitConfigError{Type: reflect.TypeOf(configuration)}
	//	}
	//
	//	var c *Configuration
	//	var found bool
	//	for i := 0; i < rv.Elem().NumField(); i++ {
	//		field := rv.Elem().Field(i)
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

func (c *Configuration) loadConfiguration(ctx context.Context, options ...ConfigurationOption) error {
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

	//TODO: c.loadStructure(c)

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

// ****** Configuration functions *********************************************

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
func configurationOptionSkipLoad() ConfigurationOption {
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
	errMissingCoreConfig   = "CC01"
	errAnonymousCoreConfig = "CC02"
)

type InvalidInitConfigError struct {
	Code string
	Type reflect.Type
}

func (e *InvalidInitConfigError) Error() string {
	switch e.Code {
	case errMissingCoreConfig:
		return "configuration error - InitConfig(core configuration not present)"
	case errAnonymousCoreConfig:
		return "configuration error - InitConfig(core configuration is anonymous)"

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
