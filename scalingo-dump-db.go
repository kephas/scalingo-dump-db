// Copyright 2019 Les porteclés de l'immobilier

// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.

// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

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

func get_db_setup(app string, env string, url_prefix string) (DbSetup, error) {
	retrieve := exec.Command("bash", "-c", fmt.Sprintf("scalingo -a %s env | grep ^%s", app, env))
	out, err := retrieve.CombinedOutput()
	check(err)

	re := regexp.MustCompile(fmt.Sprintf("%s://([^:]+):([^@]+)@[^/]+/([^?]+)([?]|$)", url_prefix))
	matches := re.FindStringSubmatch(string(out))
	if matches != nil {
		return DbSetup{User: matches[1], Password: matches[2], Database: matches[3]}, nil
	} else {
		return DbSetup{}, errors.New("impossible to parse URL")
	}
}

func dump_operation(scalingo_app string, port string, file string, url_env string, url_prefix string, cmd_maker func (DbSetup) *exec.Cmd) error {
	tunnel := exec.Command("scalingo", "db-tunnel", "-a", scalingo_app, "-p", port, url_env)
	tunnel.Start()
	//TODO wait for message in stdout to start dump

	setup,err := get_db_setup(scalingo_app, url_env, url_prefix)
	check(err)

	dump := cmd_maker(setup)

	if file == "" {
		file = default_file(scalingo_app, url_prefix)
	}
	backup,err := os.Create(file)
	check(err)

	dump.Stdout = backup
	dump.Run()
	return nil
}

func dump_postgres(scalingo_app string, port string, file string) error {
	cmd_maker := func (setup DbSetup) *exec.Cmd {
		dump := exec.Command("pg_dump", "-h", "127.0.0.1", "-p", port, "-U", setup.User, "-w", setup.Database)
		dump.Env = []string{fmt.Sprintf("PGPASSWORD=%s", setup.Password)}
		return dump
	}
	return dump_operation(scalingo_app, port, file, "SCALINGO_POSTGRESQL_URL", "postgres", cmd_maker)
}

func dump_mysql(scalingo_app string, port string, file string) error {
	cmd_maker := func (setup DbSetup) *exec.Cmd {
		dump := exec.Command("mysqldump", "-h", "127.0.0.1", "-P", port, "-u", setup.User, fmt.Sprintf("--password=%s", setup.Password), setup.Database)
		log.Printf("%v", dump)
		return dump
	}
	return dump_operation(scalingo_app, port, file, "SCALINGO_MYSQL_URL", "mysql", cmd_maker)
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
		{
			Name: "mysql",
			Usage: "dump the MySQL database",
			Action: func(c *cli.Context) error {
				return dump_mysql(scalingo_app, port, file)
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
