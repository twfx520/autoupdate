package main

import (
	"fmt"
	"log"
	"os"

	"github.com/fsnotify/fsnotify"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

type Config struct {
	Server struct {
		Port int
		Host string
	}

	AutoUpdate struct {
		LastVersion string
		Version     []struct {
			Number string
			Notes  string
			Files  []struct {
				Name string
				URL  string
			}
		}
	}
}

var config Config

func main() {

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")

	err := viper.ReadInConfig()
	if err != nil {
		log.Fatal(err)
	}
	viper.OnConfigChange(func(e fsnotify.Event) {
		fmt.Println("Config file changed:", e.Name)
		viper.Unmarshal(&config)
		fmt.Println(config)
	})
	viper.WatchConfig()

	viper.Unmarshal(&config)

	fmt.Println(config)

	staticDir, _ := os.Getwd()
	staticDir = staticDir + "/static"

	app := gin.Default()
	app.Static("/static", staticDir)
	app.GET("/version", versionHandler)
	app.Run(fmt.Sprintf("%s:%d", config.Server.Host, config.Server.Port))
}

func versionHandler(c *gin.Context) {
	c.JSON(200, config.AutoUpdate)
}
