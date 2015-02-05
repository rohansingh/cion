package main

import (
	"github.com/codegangsta/cli"
	"github.com/rohansingh/cion"
	"os"
)

func main() {
	app := cli.NewApp()

	app.Name = "cion"
	app.Usage = "commit-to-deploy system based on Docker containers"
	app.Author = ""
	app.Email = ""

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "docker",
			Usage:  "docker endpoint for running containers",
			Value:  "unix:///var/run/docker.sock",
			EnvVar: "DOCKER_HOST",
		},
		cli.StringFlag{
			Name:   "docker-cert-path",
			Usage:  "path to certificates for Docker TLS",
			EnvVar: "DOCKER_CERT_PATH",
		},
		cli.StringFlag{
			Name:   "db",
			Usage:  "path to cion.db",
			Value:  "/tmp/cion.db",
			EnvVar: "CION_DB",
		},
		cli.StringFlag{
			Name:   "github-id",
			Usage:  "github client id",
			EnvVar: "CION_GITHUB_ID",
		},
		cli.StringFlag{
			Name:   "github-secret",
			Usage:  "github client secret",
			EnvVar: "CION_GITHUB_SECRET",
		},
	}

	app.Action = func(c *cli.Context) {
		cion.Run(
			c.String("docker"),
			c.String("docker-cert-path"),
			c.String("db"),
			c.String("github-id"),
			c.String("github-secret"),
		)
	}

	app.Run(os.Args)
}
