package main

import (
	"encoding/json"
	"os"
)

type McConfig struct {
    JavaCmd string `json:"java_cmd"`
    JavaArgs string `json:"java_args"`
    JarOpts string `json:"jar_opts"`
}

type WebTLS struct {
    Enable bool `json:"enable"`
    CertFile string `json:"cert_file"`
    KeyFile string `json:"key_file"`
}

type Webconfig struct {
    Port int `json:"port"`
    TLS WebTLS `json:"tls"`
}

type Config struct {
    Minecraft McConfig  `json:"minecraft"`
    WebServer Webconfig `json:"webserver"`
}

func ReadConfig(filePath string) (*Config, error) {
    config := new(Config)

    if _, err := os.Stat(filePath); err != nil && os.IsNotExist(err) {
        config, err := LoadConfigTempl()
        if err != nil { return nil, err }
        return config, WriteConfigTempl(filePath, config)
    }

    jsonFile, err := os.Open(filePath)
    if err != nil { return nil, err }
    defer jsonFile.Close()

    err = json.NewDecoder(jsonFile).Decode(config)

    return config, err
}

func LoadConfigTempl() (*Config, error) {
var configTemplate = `{
    "minecraft": {
        "java_cmd": "java",
        "java_args": "-Xmx2048M -Xms2048M -XX:+AlwaysPreTouch -XX:+DisableExplicitGC -XX:+ParallelRefProcEnabled -XX:+PerfDisableSharedMem -XX:+UnlockExperimentalVMOptions -XX:+UseG1GC -XX:G1HeapRegionSize=8M -XX:G1HeapWastePercent=5 -XX:G1MaxNewSizePercent=40 -XX:G1MixedGCCountTarget=4 -XX:G1MixedGCLiveThresholdPercent=90 -XX:G1NewSizePercent=30 -XX:G1RSetUpdatingPauseTimePercent=5 -XX:G1ReservePercent=20 -XX:InitiatingHeapOccupancyPercent=15 -XX:MaxGCPauseMillis=200 -XX:MaxTenuringThreshold=1 -XX:SurvivorRatio=32 -Dusing.aikars.flags=https://mcflags.emc.gs -Daikars.new.flags=true",
        "jar_opts": "--nogui"
    },
    "webserver": {
        "port": 35565,
        "tls": {
            "enable": false,
            "cert_file": "",
            "key_file": ""
        }
    }
}`

    config := new(Config)
    err := json.Unmarshal([]byte(configTemplate), config)
    if err != nil { return nil, err }

    return config, nil
}

func WriteConfigTempl(filePath string, config *Config) error {
    file, err := os.Create(filePath)
    if err != nil { return err }
    defer file.Close()
    return json.NewEncoder(file).Encode(config)
}
