package config

import (
	"sync"

	"github.com/paveltrufi/mantecabox/models"
	"github.com/paveltrufi/mantecabox/utilities"
)

var (
	serverConf models.Configuration
	set        = false
	mux        = &sync.Mutex{}
)

func GetServerConf() models.Configuration {
	mux.Lock()
	defer mux.Unlock()
	if !set {
		conf, err := utilities.GetConfiguration()
		serverConf = conf
		if err != nil {
			panic(err)
		}
		set = true
	}
	return serverConf
}