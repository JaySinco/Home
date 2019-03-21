package core

import (
	"io/ioutil"
	"os"
	"sync"

	"github.com/BurntSushi/toml"
)

func Config() *TomlConf {
	globalConfig.o.Do(func() {
		file := ProjectDir() + "/test.conf"
		Info("load %s", file)
		var err error
		globalConfig.data, err = readConfig(file)
		if err != nil {
			Fatal("failed to read configure file: %s", err)
		}
	})
	return globalConfig.data
}

var globalConfig struct {
	data *TomlConf
	o    sync.Once
}

func readConfig(file string) (*TomlConf, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	var conf TomlConf
	data, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}
	if err := toml.Unmarshal(data, &conf); err != nil {
		return nil, err
	}
	return &conf, nil
}

type TomlConf struct {
	Core  CoreConf
	Metal MetalConf
}

type CoreConf struct {
	Debug         int    `toml:"LOG_DEBUG"`
	Driver        string `toml:"DATABASE_DRIVER"`
	DtbsToken     string `toml:"DATABASE_TOKEN"`
	TimerTick     string `toml:"TIMER_ENGINE_TICK"`
	TimerParallel int    `toml:"TIMER_MAX_PARALLEL"`
	TaskDeploySet string `toml:"TASK_DEPLOY_SET"`
}

type MetalConf struct {
	Sender    string `toml:"MAIL_SENDER"`
	MailToken string `toml:"MAIL_SENDER_TOKEN"`
	Recvs     string `toml:"MAIL_RECEIVERS"`
	PriceTick string `toml:"PRICE_TICK"`
}
