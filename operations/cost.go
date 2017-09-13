package operations

import (
	"fmt"
	"time"

	"github.com/evergreen-ci/sink"
	"github.com/evergreen-ci/sink/cost"
	"github.com/evergreen-ci/sink/model"
	"github.com/evergreen-ci/sink/units"
	"github.com/mongodb/amboy"
	"github.com/mongodb/grip"
	"github.com/mongodb/grip/message"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
	"golang.org/x/net/context"
)

// Cost returns the entry point for the ./sink spend sub-command,
// which has required flags.
func Cost() cli.Command {
	return cli.Command{
		Name:  "cost",
		Usage: "build cost report combining granular Evergreen and AWS data",
		Subcommands: []cli.Command{
			loadConfig(),
			collectLoop(),
			write(),
		},
	}
}

func loadConfig() cli.Command {
	return cli.Command{
		Name:  "load-config",
		Usage: "load a cost reporting configuration into the database from a file",
		Flags: dbFlags(
			cli.StringFlag{
				Name:  "file",
				Usage: "specify path to a build cost reporting config file",
			}),
		Action: func(c *cli.Context) error {
			env := sink.GetEnvironment()

			fileName := c.String("file")
			mongodbURI := c.String("dbUri")
			dbName := c.String("dbName")

			if err := configure(env, 2, true, mongodbURI, "", dbName); err != nil {
				return errors.WithStack(err)
			}

			conf, err := model.LoadCostConfig(fileName)
			if err != nil {
				return errors.WithStack(err)
			}

			if err = conf.Save(); err != nil {
				return errors.WithStack(err)
			}

			grip.Infoln("successfully saved cost configuration to database at:", mongodbURI)

			return nil
		},
	}
}

func collectLoop() cli.Command {
	return cli.Command{
		Name:  "collect",
		Usage: "collect a cost report every hour, saving the results to mongodb",
		Flags: dbFlags(costFlags()...),
		Action: func(c *cli.Context) error {
			mongodbURI := c.String("dbUri")
			dbName := c.String("dbName")
			env := sink.GetEnvironment()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			if err := configure(env, 2, false, mongodbURI, "", dbName); err != nil {
				return errors.WithStack(err)
			}

			q, err := env.GetQueue()
			if err != nil {
				return errors.Wrap(err, "problem getting queue")
			}

			if err := q.Start(ctx); err != nil {
				return errors.Wrap(err, "problem starting queue")
			}

			reports := &model.CostReports{}
			reports.Setup(env)

			amboy.IntervalQueueOperation(ctx, q, 30*time.Minute, time.Now(), true, func(queue amboy.Queue) error {
				lastHour := time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), 0, 0, 0, time.Local)

				id := fmt.Sprintf("brc-%s", lastHour)

				j := units.NewBuildCostReport(env, id)
				if err := queue.Put(j); err != nil {
					grip.Warning(err)
					return err
				}

				grip.Noticef("scheduled build cost report %s at [%s]", id, time.Now())

				numReports, _ := reports.Count()
				grip.Info(message.Fields{
					"queue":         queue.Stats(),
					"reports-count": numReports,
					"scheduled":     id,
				})
				return nil
			})

			grip.Info("process blocking indefinitely to generate reports in the background")
			select {
			case <-ctx.Done():
				// this will never fire because it never cancels.
				grip.Alert("collection terminating")
			}
		},
	}
}

func write() cli.Command {
	return cli.Command{
		Name:  "write",
		Usage: "collect and write a build cost report to a file.",
		Flags: costFlags(
			cli.StringFlag{
				Name:  "config",
				Usage: "path to configuration file, and EBS pricing information, is required",
			}),
		Action: func(c *cli.Context) error {
			start, err := time.Parse(sink.ShortDateFormat, c.String("start"))
			if err != nil {
				return errors.Wrapf(err, "problem parsing time from %s", c.String("start"))
			}
			file := c.String("config")
			dur := c.Duration("duration")

			conf, err = model.LoadCostConfig(file)
			if err != nil {
				return errors.Wrapf(err, "problem loading cost configuration from %s", file)
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			if err := writeCostReport(ctx, conf, start, dur); err != nil {
				return errors.Wrap(err, "problem writing cost report")
			}

			return nil
		},
	}
}

func writeCostReport(ctx context.Context, conf *model.CostConfig, start time.Time, dur time.Duration) error {
	duration, err := conf.GetDuration(dur)
	if err != nil {
		return errors.Wrap(err, "Problem with duration")
	}

	report, err := cost.CreateReport(ctx, start, duration, conf)
	if err != nil {
		return errors.Wrap(err, "Problem generating report")
	}

	fnDate := report.Report.Begin.Format("2006-01-02-15-04")

	filename := fmt.Sprintf("%s.%s.json", fnDate, duration)

	if err := cost.WriteToFile(conf, report, filename); err != nil {
		return errors.Wrap(err, "Problem printing report")
	}

	return nil
}