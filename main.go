package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/spf13/viper"
	"net/http"
	"os"
	"path/filepath"
)

const defaultConfigFileName = "api-proxy.conf"

func main() {
	flags := flag.NewFlagSet("flags", flag.PanicOnError)
	configPathFlag := flags.String("config", "", "Path to config file")
	flagsErr := flags.Parse(os.Args[1:])
	if nil != flagsErr {
		panic(flagsErr)
	}

	config, configError := readConfig(*configPathFlag)
	if nil != configError {
		panic(configError.Error())
	}

	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%s", config.GetString("port")),
		Handler: http.NewServeMux(),
	}

	httpServer.Handler.(*http.ServeMux).Handle("/", http.FileServer(NewWebFileSystem(config.GetString("root"))))
	httpServer.Handler.(*http.ServeMux).Handle(
		"/api/login",
		NewProxyHandler(config.GetString("api.scheme"), config.GetString("api.host"), &LoginProxyStrategy{}),
	)
	httpServer.Handler.(*http.ServeMux).Handle(
		"/api/",
		NewProxyHandler(config.GetString("api.scheme"), config.GetString("api.host"), &ApiProxyStrategy{}),
	)

	fmt.Printf("Starting web server. Port: %s. Root: '%s' \n", config.GetString("port"), config.GetString("root"))

	httpListenError := httpServer.ListenAndServe()
	if http.ErrServerClosed != httpListenError {
		panic(httpListenError)
	}
}

func readConfig(configFilePath string) (*viper.Viper, error) {
	if "" == configFilePath {
		workDir, workDirError := os.Getwd()
		if workDirError != nil {
			return nil, errors.New("Failed to determine working directory. Error: " + workDirError.Error())
		}

		configFilePath = workDir + "/" + defaultConfigFileName
	}

	// Check config file exists
	configFileInfo, configFileInfoError := os.Stat(configFilePath)
	if nil != configFileInfoError {
		return nil, errors.New(fmt.Sprintf(
			"Error while reading config file. Path: %s. Operation: %s. Error: %s",
			configFileInfoError.(*os.PathError).Path,
			configFileInfoError.(*os.PathError).Op,
			configFileInfoError.Error(),
		))
	}
	// Check config file is not a directory
	if configFileInfo.IsDir() {
		return nil, errors.New(fmt.Sprintf("Invalid config file path. %s is a directory", configFilePath))
	}
	// Check config file has supported format
	configFileExt := filepath.Ext(configFileInfo.Name())
	if !stringInSlice(configFileExt[1:], viper.SupportedExts) {
		return nil, errors.New(fmt.Sprintf("Not supported config file format. %s not supported", configFileExt))
	}

	configFileName := configFileInfo.Name()[0 : len(configFileInfo.Name())-len(configFileExt)]

	config := viper.New()
	config.AddConfigPath(filepath.Dir(configFilePath))
	config.SetConfigName(configFileName)

	configError := config.ReadInConfig()
	if nil != configError {
		return nil, configError
	}

	return config, nil
}
