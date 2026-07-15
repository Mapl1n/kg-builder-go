package config

import (
	"fmt"
	"os"
	"strconv"
)
type Config struct {
	ServerPort, ESHost, ESPort, ESIndex, TikaURL string
}
func Load() *Config {
	return &Config{
		ServerPort: getEnv("SERVER_PORT", "8083"),
		ESHost:     getEnv("ES_HOST", "localhost"),
		ESPort:     getEnv("ES_PORT", "9200"),
		ESIndex:    getEnv("ES_INDEX", "knowledge_graph"),
		TikaURL:    getEnv("TIKA_URL", "http://localhost:9998"),
	}
}
func (c *Config) ESURL() string { return fmt.Sprintf("http://%s:%s", c.ESHost, c.ESPort) }
func getEnv(k, d string) string { if v := os.Getenv(k); v != "" { return v }; return d }
func getEnvInt(k string, d int) int { if v := os.Getenv(k); v != "" { if n, e := strconv.Atoi(v); e == nil { return n } }; return d }
