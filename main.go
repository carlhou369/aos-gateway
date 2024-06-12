package main

import (
	"context"
	"fmt"
	"gateway/db"
	"gateway/log"
	"gateway/rpc"
	"gateway/trie"
	"io/ioutil"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	cli "gopkg.in/urfave/cli.v1"
	yaml "gopkg.in/yaml.v2"
)

var (
	OriginCommandHelpTemplate = `{{.Name}}{{if .Subcommands}} command{{end}}{{if .Flags}} [command options]{{end}} {{.ArgsUsage}}
{{if .Description}}{{.Description}}
{{end}}{{if .Subcommands}}
SUBCOMMANDS:
  {{range .Subcommands}}{{.Name}}{{with .ShortName}}, {{.}}{{end}}{{ "\t" }}{{.Usage}}
  {{end}}{{end}}{{if .Flags}}
OPTIONS:
{{range $.Flags}}   {{.}}
{{end}}
{{end}}`
)
var app *cli.App

var (
	configPathFlag = cli.StringFlag{
		Name:  "config",
		Usage: "config path",
		Value: "./config.yml",
	}
	logLevelFlag = cli.IntFlag{
		Name:  "log",
		Usage: "log level",
		Value: log.InfoLog,
	}
	logFilePath = cli.StringFlag{
		Name:  "logPath",
		Usage: "log root path",
		Value: "./logs",
	}
)

func init() {
	app = cli.NewApp()
	app.Version = "v1.0.0"
	app.Commands = []cli.Command{
		commandStart,
	}

	cli.CommandHelpTemplate = OriginCommandHelpTemplate
}

func main() {
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

var commandStart = cli.Command{
	Name:  "start",
	Usage: "start loading contract gas fee",
	Flags: []cli.Flag{
		configPathFlag,
		logLevelFlag,
		logFilePath,
	},
	Action: Start,
}

type ProxyConfig struct {
	Port             string              `yaml:"port"`
	OpenAiKey        []string            `yaml:"openai_key"`
	MaxPendingLength int                 `yaml:"max_pending"`
	Host             string              `yaml:"host"`
	ModelConfig      map[string][]string `yaml:"bs_model"`
	MongoURI         string              `yaml:"mongo_uri"`
	Sensitive        string              `yaml:"sensitive"`
}

func Start(ctx *cli.Context) {
	defer func() {
		db.MgoCli.Disconnect(context.Background())
	}()
	logLevel := ctx.Int(logLevelFlag.Name)
	fmt.Println("log level", logLevel)
	logPath := ctx.String(logFilePath.Name)

	filename := fmt.Sprintf("/gateway_%v.log", strings.ReplaceAll(time.Now().Format("2006-01-02 15:04:05"), " ", "_"))
	fmt.Println("log file path", logPath+filename)
	err := os.MkdirAll(logPath, 0777)
	if err != nil {
		panic(err)
	}
	logFile, err := os.Create(logPath + filename)
	if err != nil {
		panic(err)
	}
	defer logFile.Close()

	log.InitLog(log.DebugLog, logFile)

	conf := loadConfig(ctx)
	if conf.Host != "" {
		rpc.Host = conf.Host
	}

	//load senstive word
	err = trie.LoadSensitive(conf.Sensitive)
	if err != nil {
		panic(err)
	}

	//init db
	db.MongoURI = conf.MongoURI
	db.Init()

	rpc.InitRpcService(conf.Port, conf.OpenAiKey, conf.MaxPendingLength, conf.ModelConfig)

	contx := context.Background()
	err = rpc.RpcServer.Start(contx)
	if err != nil {
		log.Fatal(err)
	}
	waitToExit()
}

func loadConfig(ctx *cli.Context) ProxyConfig {
	var gatewayConfig ProxyConfig
	if ctx.IsSet(configPathFlag.Name) {
		configPath := ctx.String(configPathFlag.Name)
		b, err := ioutil.ReadFile(configPath)
		if err != nil {
			log.Fatal("read config error", err)
		}
		err = yaml.Unmarshal(b, &gatewayConfig)
		if err != nil {
			log.Fatal(err)
		}
	}
	return gatewayConfig
}

func waitToExit() {
	exit := make(chan bool, 0)
	sc := make(chan os.Signal, 1)
	if !signal.Ignored(syscall.SIGHUP) {
		signal.Notify(sc, syscall.SIGHUP)
	}
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		for sig := range sc {
			fmt.Printf("received exit signal:%v", sig.String())
			close(exit)
			break
		}
	}()
	<-exit
}
