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

func panic_if(e error) {
	if e != nil {
		panic(e)
	}
}

func debug(setup Setup, object interface{}) {
	if setup.Debug {
		log.Printf("%v", object)
	}
}

func default_file(app string, db string) string {
	return fmt.Sprintf("%s_%s_%s.bak", app, db, time.Now().Format(time.RFC3339))
}

type Setup struct {
	App string
	Port string
	File string
	Debug bool
	User string
	Password string
	Database string
}

func get_db_setup(setup Setup, env string, url_prefix string) (Setup, error) {
	retrieve := exec.Command("bash", "-c", fmt.Sprintf("scalingo -a %s env | grep ^%s", setup.App, env))
	out, err := retrieve.CombinedOutput()
	panic_if(err)

	re := regexp.MustCompile(fmt.Sprintf("%s://([^:]+):([^@]+)@[^/]+/([^?]+)([?]|$)", url_prefix))
	matches := re.FindStringSubmatch(string(out))
	if matches != nil {
		setup.User = matches[1]
		setup.Password = matches[2]
		setup.Database = matches[3]
		return setup, nil
	} else {
		return setup, errors.New("impossible to parse URL")
	}
}

func dump_operation(setup Setup, url_env string, url_prefix string, cmd_maker func (Setup) *exec.Cmd) error {
	tunnel := exec.Command("scalingo", "db-tunnel", "-a", setup.App, "-p", setup.Port, url_env)
	debug(setup, tunnel)
	tunnel.Start()
	//TODO wait for message in stdout to start dump

	setup,err := get_db_setup(setup, url_env, url_prefix)
	panic_if(err)

	dump := cmd_maker(setup)
	debug(setup, dump)
	if setup.File == "" {
		setup.File = default_file(setup.App, url_prefix)
	}
	backup,err := os.Create(setup.File)
	panic_if(err)

	dump.Stdout = backup
	dump.Run()
	// if err = tunnel.Process.Kill(); err != nil {
	// 	log.Printf("couldn't kill tunnel; %v" , err)
	//}
	return nil
}

func dump_postgres(setup Setup) error {
	cmd_maker := func (setup Setup) *exec.Cmd {
		dump := exec.Command("pg_dump", "-h", "127.0.0.1", "-p", setup.Port, "-U", setup.User, "-w", setup.Database)
		dump.Env = []string{fmt.Sprintf("PGPASSWORD=%s", setup.Password)}
		return dump
	}
	return dump_operation(setup, "SCALINGO_POSTGRESQL_URL", "postgres", cmd_maker)
}

func dump_mysql(setup Setup) error {
	cmd_maker := func (setup Setup) *exec.Cmd {
		dump := exec.Command("mysqldump", "-h", "127.0.0.1", "-P", setup.Port, "-u", setup.User, fmt.Sprintf("--password=%s", setup.Password), setup.Database)
		return dump
	}
	return dump_operation(setup, "SCALINGO_MYSQL_URL", "mysql", cmd_maker)
}



func main () {
	var setup Setup

	app := cli.NewApp()
	app.Name = "Scalingo Database dumper"
	app.Version = "0.1"
	app.Flags = []cli.Flag{
		cli.StringFlag{Name: "app, a", Destination: &setup.App},
		cli.StringFlag{Name: "port, p", Value: "31415", Destination: &setup.Port},
		cli.StringFlag{Name: "file, f", Destination: &setup.File},
		cli.BoolFlag{Name: "debug, d", Destination: &setup.Debug},
	}

	app.Commands = []cli.Command {
		{
			Name: "pg",
			Aliases: []string{"postgres", "postgresql"},
			Usage: "dump the PostgreSQL database",
			Action: func(c *cli.Context) error {
				return dump_postgres(setup)
			},
		},
		{
			Name: "mysql",
			Usage: "dump the MySQL database",
			Action: func(c *cli.Context) error {
				return dump_mysql(setup)
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
