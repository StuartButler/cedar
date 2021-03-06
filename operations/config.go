package operations

import (
	"github.com/evergreen-ci/cedar"
	"github.com/evergreen-ci/cedar/model"
	"github.com/evergreen-ci/cedar/util"
	"github.com/mongodb/grip"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

func dumpCedarConfig() cli.Command {
	return cli.Command{
		Name:  "dump",
		Usage: "write current cedar application configuration to a file",
		Flags: dbFlags(
			cli.StringFlag{
				Name:  "file",
				Usage: "specify path to a cedar application config file",
			}),
		Action: func(c *cli.Context) error {
			env := cedar.GetEnvironment()

			fileName := c.String("file")
			mongodbURI := c.String(dbURIFlag)
			dbName := c.String(dbNameFlag)

			if err := configure(env, 2, true, mongodbURI, "", dbName); err != nil {
				return errors.WithStack(err)
			}

			conf := &model.CedarConfig{}
			conf.Setup(env)

			if err := conf.Find(); err != nil {
				return errors.WithStack(err)
			}

			return errors.WithStack(util.WriteJSON(fileName, conf))
		},
	}
}

func loadCedarConfig() cli.Command {
	return cli.Command{
		Name:  "load",
		Usage: "loads cedar application configuration from a file",
		Flags: dbFlags(
			cli.StringFlag{
				Name:  "file",
				Usage: "specify path to a cedar application config file",
			}),
		Action: func(c *cli.Context) error {
			env := cedar.GetEnvironment()

			fileName := c.String("file")
			mongodbURI := c.String(dbURIFlag)
			dbName := c.String(dbNameFlag)

			if err := configure(env, 2, true, mongodbURI, "", dbName); err != nil {
				return errors.WithStack(err)
			}

			conf, err := model.LoadCedarConfig(fileName)
			if err != nil {
				return errors.WithStack(err)
			}
			conf.Setup(env)

			if err = conf.Save(); err != nil {
				return errors.WithStack(err)
			}

			grip.Infoln("successfully application configuration to database at:", mongodbURI)
			return nil
		},
	}
}
