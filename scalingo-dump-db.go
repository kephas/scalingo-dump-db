package main

import (
	"errors"
	"fmt"
	"github.com/urfave/cli"
	"log"
	"os"
	"os/exec"
	"regexp"
	"time"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func default_file(app string, db string) string {
	return fmt.Sprintf("%s_%s_%s.bak", app, db, time.Now().Format(time.RFC3339))
}

type DbSetup struct {
	User string
	Password string
	Database string
}


func get_postgres_setup(app string) (DbSetup, error) {
	retrieve := exec.Command("bash", "-c", fmt.Sprintf("scalingo -a %s env | grep ^SCALINGO_POSTGRESQL_URL", app))
	out, err := retrieve.CombinedOutput()
	check(err)

	re := regexp.MustCompile("postgres://([^:]+):([^@]+)@[^/]+/([^?]+)([?]|$)")
	matches := re.FindStringSubmatch(string(out))
	if matches != nil {
		return DbSetup{User: matches[1], Password: matches[2], Database: matches[3]}, nil
	} else {
		return DbSetup{}, errors.New("impossible to parse URL")
	}
}

func dump_postgres(scalingo_app string, port string, file string) error {
	tunnel := exec.Command("scalingo", "db-tunnel", "-a", scalingo_app, "-p", port, "SCALINGO_POSTGRESQL_URL")
	tunnel.Start()
	//TODO wait for message in stdout to start dump

	setup,err := get_postgres_setup(scalingo_app)
	check(err)

	dump := exec.Command("pg_dump", "-h", "127.0.0.1", "-p", port, "-U", setup.User, "-w", setup.Database)
	dump.Env = []string{fmt.Sprintf("PGPASSWORD=%s", setup.Password)}

	if file == "" {
		file = default_file(scalingo_app, "pg")
	}
	backup,err := os.Create(file)
	check(err)

	dump.Stdout = backup
	dump.Run()
	return nil
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
				return dump_postgres(scalingo_app, port, file)
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
