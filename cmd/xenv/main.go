package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/ionrock/xenv/config"
	"github.com/urfave/cli"
)

var builddate = ""
var gitref = ""

func XeAction(c *cli.Context) error {
	fmt.Println("loading " + c.String("config"))

	cfgs, err := config.NewXeConfig(c.String("config"))
	if err != nil {
		fmt.Printf("error loading config: %s\n", err)
	}

	env := config.NewEnvironment(filepath.Dir(c.String("config")), cfgs)

	for _, cfg := range cfgs {
		handler := env.ConfigHandler
		if c.Bool("data") {
			handler = env.DataHandler
		}
		if err := handler(cfg); err != nil {
			return err
		}
	}

	parts := c.Args()

	fmt.Printf("Going to start: %s\n", parts)
	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	cmd.Env = env.Config.ToEnv()

	err = cmd.Run()
	env.CleanUp()

	if err != nil {
		return err
	}

	return nil
}

func main() {
	app := cli.NewApp()

	app.Version = fmt.Sprintf("%s-%s", gitref, builddate)

	app.Name = "xe"
	app.Usage = "Start and monitor processes creating an executable environment."
	app.ArgsUsage = "[COMMAND]"
	app.Action = XeAction

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "config, c",
			Usage: "Path to the xe config file, default is ./xe.yml",
			Value: "xe.yml",
		},

		cli.BoolFlag{
			Name:  "data, d",
			Usage: "Only compute the data and print it out.",
		},
	}

	app.Run(os.Args)
}
