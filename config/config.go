package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
)

const (
	DEFAULT_LOG_LEVEL        = 2
	DEFAULT_CONFIG_FILE_NAME = "./config.json"
)

var DefConfig *Config

//Config object used by ontology-instance
type Config struct {
	JsonRpcAddress   string            `json:"json_rpc_address"`
	FlashPoolAddress string            `json:"flash_pool_address"`
	AssetMap         map[string]string `json:"asset_map"`
	ScanInterval     uint64            `json:"scan_interval"`
}

func NewConfig(fileName string) (*Config, error) {
	data, err := ioutil.ReadFile(fileName)
	if err != nil {
		return nil, err
	}
	cfg := &Config{}
	err = json.Unmarshal(data, cfg)
	if err != nil {
		return nil, fmt.Errorf("json.Unmarshal Config:%s error:%s", data, err)
	}
	return cfg, nil
}
