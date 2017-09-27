package evergreen

import (
	"encoding/json"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/mongodb/grip"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
)

// Distro holds information for a single distro within a host.
type Distro struct {
	DistroID string `json:"_id"`
}

// DistroCost holds full cost and provider information for a distro.
type DistroCost struct {
	DistroID         string        `json:"distro_id"`
	Provider         string        `json:"provider"`
	InstanceType     string        `json:"instance_type,omitempty"`
	SumTimeTaken     time.Duration `json:"sum_time_taken"`
	SumEstimatedCost float64       `json:"sum_estimated_cost"`
}

// GetDistros is a wrapper function of get for getting all distros from the
// Evergreen API.
func (c *Client) GetDistros(ctx context.Context) ([]*Distro, error) {
	data, link, err := c.get(ctx, "/distros")
	if link != "" {
		return nil, errors.New("/distros should not be a paginated route")
	}
	if err != nil {
		return nil, errors.Wrap(err, "error in getting distros")
	}
	distros := []*Distro{}
	if err := json.Unmarshal(data, &distros); err != nil {
		return nil, err
	}
	return distros, nil
}

// GetDistroCost is a wrapper function of get for getting all distro costs
// from the evergreen API given a distroID.
func (c *Client) GetDistroCost(ctx context.Context, distroID, starttime, duration string) (*DistroCost, error) {
	data, link, err := c.get(ctx, "/cost/distro/"+distroID+"?starttime="+starttime+"&duration="+duration)
	if link != "" {
		return nil, errors.New("/cost/distro should not be a paginated route")
	}
	if err != nil {
		return nil, errors.Wrap(err, "error in GetDistroCost")
	}
	distro := &DistroCost{}
	if err := json.Unmarshal(data, &distro); err != nil {
		return nil, err
	}
	return distro, nil
}

// A helper function for GetEvergreenDistrosData that gets distroID of
// all distros by calling GetDistros.
func (c *Client) getDistroIDs(ctx context.Context) ([]string, error) {
	distroIDs := []string{}
	distros, err := c.GetDistros(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "error getting distros ids")
	}
	for _, d := range distros {
		distroIDs = append(distroIDs, d.DistroID)
	}
	return distroIDs, nil
}

// A helper function for GetEvergreenDistrosData that gets provider,
// instance type, and total time for a given list of distros found.
func (c *Client) getDistroCosts(ctx context.Context, distroIDs []string, st, dur string) ([]*DistroCost, error) {
	distroCosts := []*DistroCost{}
	costs := make(chan *DistroCost)
	distros := make(chan string, len(distroIDs))
	catcher := grip.NewCatcher()
	wg := &sync.WaitGroup{}

	for _, idx := range rand.Perm(len(distroIDs)) {
		distros <- distroIDs[idx]
	}
	close(distros)

	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for distro := range distros {
				if ctx.Err() != nil {
					return
				}

				dc, err := c.GetDistroCost(ctx, distro, st, dur)
				catcher.Add(errors.Wrap(err, "error when getting distro cost data from Evergreen"))
				if dc == nil {
					continue
				}

				costs <- dc
			}
		}()
	}

	go func() {
		wg.Wait()
		close(costs)
	}()

	for evgdc := range costs {
		if evgdc.SumTimeTaken > 0 {
			distroCosts = append(distroCosts, evgdc)
		}
	}

	if catcher.HasErrors() {
		return nil, catcher.Resolve()
	}

	return distroCosts, nil
}

// GetEvergreenDistrosData retrieves distros cost data from Evergreen.
func (c *Client) GetEvergreenDistrosData(ctx context.Context, starttime time.Time, duration time.Duration) ([]*DistroCost, error) {
	st := starttime.Format("2006-01-02T15:04:05Z07:00")
	dur := strings.TrimRight(duration.String(), "0s")

	distroIDs, err := c.getDistroIDs(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "error in getting distroID in GetEvergreenDistrosData")
	}

	distroCosts, err := c.getDistroCosts(ctx, distroIDs, st, dur)
	if err != nil {
		return nil, errors.Wrap(err, "error in getting distro costs in GetEvergreenDistrosData")
	}

	return distroCosts, nil
}
