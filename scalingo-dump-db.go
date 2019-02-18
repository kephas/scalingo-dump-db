package main

import (
	"fmt"
	"github.com/urfave/cli"
	"log"
	"os"
	"os/exec"
	"time"
)

func default_file(app string, db string) string {
	return fmt.Sprintf("%s_%s_%s.bak", app, db, time.Now().Format(time.RFC3339))
}

func main () {
	var scalingo_app, port, file string

	app := cli.NewApp()
	app.Name = "Scalingo Database dumper"
	app.Version = "0.1"
	app.Flags = []cli.Flag{
		cli.StringFlag{Name: "app, a", Destination: &scalingo_app},
		cli.StringFlag{Name: "port, p", Value: "31415", Destination: &port},
		cli.StringFlag{Name: "file, f", Destination: &file},
	}

	app.Commands = []cli.Command {
		{
			Name: "pg",
			Aliases: []string{"postgres", "postgresql"},
			Usage: "dump the PostgreSQL database",
			Action: func(c *cli.Context) error {
				tunnel := exec.Command("scalingo", "db-tunnel", "-a", scalingo_app, "-p", port, "SCALINGO_POSTGRESQL_URL")
				tunnel.Start()

				return nil
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
