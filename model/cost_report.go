package model

import (
	"fmt"
	"time"

	"github.com/evergreen-ci/sink"
	"github.com/evergreen-ci/sink/bsonutil"
	"github.com/mongodb/grip"
	"github.com/mongodb/grip/message"
	"github.com/pkg/errors"
	"github.com/tychoish/anser/db"
)

const costReportCollection = "buildCostReports"

// CostReport provides the structure for the report that will be returned for
// the build cost reporting tool.
type CostReport struct {
	ID        string             `bson:"_id" json:"-" yaml:"-"`
	Report    CostReportMetadata `bson:"report" json:"report" yaml:"report"`
	Evergreen EvergreenCost      `bson:"evergreen" json:"evergreen" yaml:"evergreen"`
	Providers []CloudProvider    `bson:"providers" json:"providers" yaml:"providers"`

	env       sink.Environment
	populated bool
}

var (
	costReportReportKey    = bsonutil.MustHaveTag(CostReport{}, "Report")
	costReportEvergreenKey = bsonutil.MustHaveTag(CostReport{}, "Evergreen")
	costReportProvidersKey = bsonutil.MustHaveTag(CostReport{}, "Providers")
)

func (r *CostReport) Setup(e sink.Environment) { r.env = e }
func (r *CostReport) IsNil() bool              { return r.populated }
func (r *CostReport) FindID(id string) error {
	conf, session, err := sink.GetSessionWithConfig(r.env)
	if err != nil {
		return errors.WithStack(err)
	}
	defer session.Close()

	r.populated = false

	err = session.DB(conf.DatabaseName).C(costReportCollection).FindId(id).One(r)
	if db.ResultsNotFound(err) {
		return errors.Errorf("could not find cost reporting document %s in the database", id)
	} else if err != nil {
		errors.Wrap(err, "problem finding cost config document")
	}
	r.populated = true

	return nil
}

func (r *CostReport) Save() error {
	// TOOD call some kind of validation routine to avoid saving junk data
	conf, session, err := sink.GetSessionWithConfig(r.env)
	if err != nil {
		return errors.WithStack(err)
	}
	defer session.Close()

	changeInfo, err := session.DB(conf.DatabaseName).C(costReportCollection).UpsertId(r.ID, r)
	grip.Debug(message.Fields{
		"ns":          fmt.Sprintf("%s.%s", conf.DatabaseName, costReportCollection),
		"id":          r.ID,
		"operation":   "save build cost report",
		"change-info": changeInfo,
	})
	if db.ResultsNotFound(err) {
		return errors.New("could not find cost reporting document in the database")
	}

	return errors.Wrap(err, "problem saving cost reporting configuration")
}

// Report provides time information on the overall structure.
type CostReportMetadata struct {
	Generated time.Time `bson:"generated" json:"generated" yaml:"generated"`
	Begin     time.Time `bson:"begin" json:"begin" yaml:"begin"`
	End       time.Time `bson:"end" json:"end" yaml:"end"`
}

var (
	costReportMetadataGeneratedKey = bsonutil.MustHaveTag(CostReportMetadata{}, "Generated")
	costReportMetadataBeginKey     = bsonutil.MustHaveTag(CostReportMetadata{}, "Begin")
	costReportMetadataEndKey       = bsonutil.MustHaveTag(CostReportMetadata{}, "End")
)

// Evergreen provides a list of the projects and distros in Evergreen.
type EvergreenCost struct {
	Projects []EvergreenProjectCost `bson:"projects" json:"projects" yaml:"projects"`
	Distros  []EvergreenDistroCost  `bson:"distros" json:"distros" yaml:"distros"`
}

var (
	costReportEvergreenCostProjectsKey = bsonutil.MustHaveTag(EvergreenCost{}, "Projects")
	costReportEvergreenCostDistroskey  = bsonutil.MustHaveTag(EvergreenCost{}, "Distros")
)

// EvergreenProjectCost holds the name and tasks for a single project.
type EvergreenProjectCost struct {
	Name  string              `bson:"name" json:"name" yaml:"name"`
	Tasks []EvergreenTaskCost `bson:"tasks" json:"tasks" yaml:"tasks"`
}

var (
	costReportEvergreenProjectCostNameKey  = bsonutil.MustHaveTag(EvergreenProjectCost{}, "Name")
	costReportEvergreenProjectCostTaskskey = bsonutil.MustHaveTag(EvergreenProjectCost{}, "Tasks")
)

// EvergreenDistro holds the information for a single distro in Evergreen.
type EvergreenDistroCost struct {
	Name            string `bson:"name" json:"name" yaml:"name"`
	Provider        string `bson:"provider" json:"provider" yaml:"provider"`
	InstanceType    string `bson:"instance_type,omitempty" json:"instance_type,omitempty" yaml:"instance_type,omitempty"`
	InstanceSeconds int64  `bson:"instance_seconds,omitempty" json:"instance_seconds,omitempty" yaml:"instance_seconds,omitempty"`
}

var (
	costReportEvergreenDistroNameKey            = bsonutil.MustHaveTag(EvergreenDistroCost{}, "Name")
	costReportEvergreenDistroProviderKey        = bsonutil.MustHaveTag(EvergreenDistroCost{}, "Provider")
	costReportEvergreenDistroInstanceTypeKey    = bsonutil.MustHaveTag(EvergreenDistroCost{}, "InstanceType")
	costReportEvergreenDistroInstanceSecondsKey = bsonutil.MustHaveTag(EvergreenDistroCost{}, "InstanceSeconds")
)

// Task holds the information for a single task within a project.
type EvergreenTaskCost struct {
	Githash      string `bson:"githash" json:"githash" yaml:"githash"`
	Name         string `bson:"name" json:"name" yaml:"name"`
	Distro       string `bson:"distro" json:"distro" yaml:"distro"`
	BuildVariant string `bson:"variant" json:"variant" yaml:"variant"`
	TaskSeconds  int64  `bson:"seconds" json:"seconds" yaml:"seconds"`
}

var (
	costReportEvergreenTaskCostGithashKey     = bsonutil.MustHaveTag(EvergreenTaskCost{}, "Githash")
	costReportEvergreenTaskCostNameKey        = bsonutil.MustHaveTag(EvergreenTaskCost{}, "Name")
	costReportEvergreenTaskCostDistroKey      = bsonutil.MustHaveTag(EvergreenTaskCost{}, "Distro")
	costReportEvergreenTaskCostBuildVarianKey = bsonutil.MustHaveTag(EvergreenTaskCost{}, "BuildVariant")
	costReportEvergreenTaskCostSecondKey      = bsonutil.MustHaveTag(EvergreenTaskCost{}, "TaskSeconds")
)

// Provider holds account information for a single provider.
type CloudProvider struct {
	Name     string         `bson:"name" json:"name" yaml:"name"`
	Accounts []CloudAccount `bson:"accounts" json:"accounts" yaml:"accounts"`
	Cost     float32        `bson:"cost" json:"cost" yaml:"cost"`
}

var (
	costReportCloudProviderNameKey     = bsonutil.MustHaveTag(CloudProvider{}, "Name")
	costReportCloudProviderAccountsKey = bsonutil.MustHaveTag(CloudProvider{}, "Accounts")
	costReportCloudProviderCostKey     = bsonutil.MustHaveTag(CloudProvider{}, "Cost")
)

// Account holds the name and services of a single account for a provider.
type CloudAccount struct {
	Name     string           `bson:"name" json:"name" yaml:"name"`
	Services []AccountService `bson:"services" json:"services" yaml:"services"`
}

var (
	costReportCloudAccountNameKey     = bsonutil.MustHaveTag(CloudAccount{}, "Name")
	costReportCloudAccountServicesKey = bsonutil.MustHaveTag(CloudAccount{}, "Services")
)

// Service holds the item information for a single service within an account.
type AccountService struct {
	Name  string        `bson:"name" json:"name" yaml:"name"`
	Items []ServiceItem `bson:"items" json:"items" yaml:"items"`
	Cost  float32       `bson:"cost" json:"cost" yaml:"cost"`
}

var (
	costReportAccountServiceNameKey  = bsonutil.MustHaveTag(AccountService{}, "Name")
	costReportAccountServiceItemsKey = bsonutil.MustHaveTag(AccountService{}, "Items")
	costReportAccountServiceCostKey  = bsonutil.MustHaveTag(AccountService{}, "Cost")
)

// Item holds the information for a single item for a service.
type ServiceItem struct {
	Name       string  `bson:"name" json:"name" yaml:"name"`
	ItemType   string  `bson:"type" json:"type" yaml:"type"`
	Launched   int     `bson:"launched" json:"launched" yaml:"launched"`
	Terminated int     `bson:"terminated" json:"terminated" yaml:"terminated"`
	FixedPrice float32 `bson:"fixed_price,omitempty" json:"fixed_price,omitempty" yaml:"fixed_price,omitempty"`
	AvgPrice   float32 `bson:"avg_price,omitempty" json:"avg_price,omitempty" yaml:"avg_price,omitempty"`
	AvgUptime  float32 `bson:"avg_uptime,omitempty" json:"avg_uptime,omitempty" yaml:"avg_uptime,omitempty"`
	TotalHours int     `bson:"total_hors,omitempty" json:"total_hors,omitempty" yaml:"total_hors,omitempty"`
}

var (
	costReportServiceItemNameKey       = bsonutil.MustHaveTag(ServiceItem{}, "Name")
	costReportServiceItemItemTpyeKey   = bsonutil.MustHaveTag(ServiceItem{}, "ItemType")
	costReportServiceItemLaunchedKey   = bsonutil.MustHaveTag(ServiceItem{}, "Launched")
	costReportServiceItemTerminatedKey = bsonutil.MustHaveTag(ServiceItem{}, "Terminated")
	costReportServiceItemFixedPriceKey = bsonutil.MustHaveTag(ServiceItem{}, "FixedPrice")
	costReportServiceItemAvgPriceKey   = bsonutil.MustHaveTag(ServiceItem{}, "AvgPrice")
	costReportServiceItemAvgUptimeKey  = bsonutil.MustHaveTag(ServiceItem{}, "AvgUptime")
	costReportServiceItemTotalHoursKey = bsonutil.MustHaveTag(ServiceItem{}, "TotalHours")
)
