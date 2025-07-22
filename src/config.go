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

type Webconfig struct {
    Port int `json:"port"`
}

type Config struct {
    Minecraft McConfig  `json:"minecraft"`
    Webserver Webconfig `json:"webserver"`
}

func ReadConfig(file string) (*Config, error) {
    var config *Config
    if file == "" {
        config, err := LoadConfigTempl()
        if err != nil { return nil, err }
        return config, WriteConfigTempl(config)
    }

    jsonFile, err := os.Open(file)
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
        "port": 35565
    }
}`

    config := new(Config)
    err := json.Unmarshal([]byte(configTemplate), config)
    if err != nil { return nil, err }

    return config, nil
}

func WriteConfigTempl(config *Config) error {
    jsonFile, err := os.Create("config.json")
    if err != nil { return err }
    defer jsonFile.Close()
    return json.NewEncoder(jsonFile).Encode(config)
}
