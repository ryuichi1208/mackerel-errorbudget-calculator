package lib

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/mackerelio/mackerel-client-go"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	mp "github.com/mackerelio/go-mackerel-plugin"
)

// Name is executable name of this application.
const Name string = "mackerel-plugin-conntrack"

// Version is version string of this application.
const Version string = "0.1.0"

var logger *zap.Logger
var opts options

type options struct {
	Origization      string  `short:"o" long:"org" description:"" required:"false"`
	Service          string  `short:"s" long:"service" description:"" required:"true"`
	Roles            string  `short:"r" long:"roles" description:"" required:"true"`
	Filter           string  `short:"f" long:"filter" description:"" required:"false"`
	Metrics          string  `short:"m" long:"metrics" description:"" required:"true"`
	ObjectiveLatency float64 `long:"objective-latency" description:"Specify latency in the range of float64" default:"100" required:"true"`
	ErrorBudgetSize  float64 `long:"errorbudget-size" description:"Specify SLO value 0.0 ~ 100.0" default:"99.9" required:"false"`
	TimeWindow       int     `long:"timewindow" description:"Specify time window (unit: day)" default:"30" required:"false"`
	Prefix           string  `long:"prefix" description:"" required:"false"`
	Version          bool    `long:"version" description:"print version and exit" required:"false"`
	Debug            bool    `long:"debug" description:"Enable debug mode" required:"false"`
}

type Mackerel struct {
	client  mackerel.Client
	org     string
	service string
	roles   string
	filter  string
	metrics string
}

type SLO struct {
	ObjectivesLatency float64
	TimeWindow        int
	ErrorBudgetSize   float64
}

func NewMackerel(token, org, service, roles, filter, metrics string) Mackerel {
	return Mackerel{
		client:  *mackerel.NewClient(token),
		org:     org,
		service: service,
		roles:   roles,
		filter:  filter,
		metrics: metrics,
	}
}

func (m *Mackerel) FetchHosts() ([]string, error) {
	var hosts []string
	t, err := m.client.FindHosts(&mackerel.FindHostsParam{
		Service: m.service,
		Roles:   []string{m.roles},
	})

	if err != nil {
		return hosts, err
	}

	if m.filter != "" {
		for _, v := range t {
			if strings.Contains(v.Name, m.filter) {
				hosts = append(hosts, v.ID)
			}
		}
	} else {
		for _, v := range t {
			hosts = append(hosts, v.ID)
		}
	}

	return hosts, nil
}

func (m *Mackerel) FetchMetrics(res chan<- map[time.Time]float64, hostId string) error {
	dt := time.Now()
	from_unix := dt.Unix()
	to_unix := dt.Unix()

	s := make(map[time.Time]float64)
	for i := 1; i <= opts.TimeWindow*2; i++ {
		from_unix = dt.Add(-12 * time.Hour * time.Duration(i)).Unix()
		metrics, err := m.client.FetchHostMetricValues(hostId, m.metrics, from_unix, to_unix)
		if err != nil {
			return err
		}

		for _, m := range metrics {
			dtFromUnix := time.Unix(m.Time, 0)
			s[dtFromUnix] = m.Value.(float64)
		}

		to_unix = from_unix
	}

	res <- s

	return nil
}

func (m Mackerel) ObjecttiveEvaluation(res map[time.Time]float64, slo SLO) (float64, float64, float64) {
	var violationsCount float64
	for k, v := range res {
		if v > slo.ObjectivesLatency {
			logger.Debug("over", zap.String("ObjecttiveEvaluation", fmt.Sprintf("%s: %f", k, v)))
			violationsCount++
		} else {
			logger.Debug("under", zap.String("ObjecttiveEvaluation", fmt.Sprintf("%s: %f", k, v)))
		}
	}

	logger.Debug("msg", zap.String("ObjecttiveEvaluation", fmt.Sprintf("%d: %f perc: %f", len(res), violationsCount, (100-(violationsCount/(float64(len(res)))*100)))))

	return violationsCount, (100 - (violationsCount / (float64(len(res))) * 100)), float64(len(res)) - ((float64(len(res)) * opts.ErrorBudgetSize) / 100) - violationsCount
}

func calcAverage(target []map[time.Time]float64) map[time.Time]float64 {
	var avg float64
	var result map[time.Time]float64

	if len(target) > 0 {
		result = make(map[time.Time]float64, 0)
		for date := range target[0] {
			for _, host := range target {
				t, ok := host[date]
				if ok {
					avg += t
				}
			}
			result[date] = avg / float64(len(target))
			avg = 0
		}

	}

	return result
}

func run() error {
	cli := NewMackerel(getMackerelToke(), opts.Origization, opts.Service, opts.Roles, opts.Filter, opts.Metrics)
	a, err := cli.FetchHosts()
	if err != nil {
		return err
	}

	if len(a) < 1 {
		return fmt.Errorf("Not Fetch hosts")
	}

	logKey := "key"

	c := make(chan map[time.Time]float64, len(a))
	eg := new(errgroup.Group)
	for _, v := range a {
		v := v
		eg.Go(func() error {
			return cli.FetchMetrics(c, v)
		})
	}
	if err := eg.Wait(); err != nil {
		fmt.Println(err)
	}

	res := make([]map[time.Time]float64, 0)
	for i, v := range a {
		logger.Debug("[DEBUG]", zap.String(logKey, fmt.Sprintf("%d:%s", i, v)))
		res = append(res, <-c)
	}
	slo := SLO{
		ObjectivesLatency: opts.ObjectiveLatency,
		TimeWindow:        opts.TimeWindow,
		ErrorBudgetSize:   opts.ErrorBudgetSize,
	}
	logger.Debug("msg options", zap.String("", fmt.Sprintf("%f %d %f", opts.ObjectiveLatency, opts.TimeWindow, opts.ErrorBudgetSize)))

	vioCount, budget, budgetSize := cli.ObjecttiveEvaluation(calcAverage(res), slo)

	b := NewBudget(vioCount, budget, budgetSize)
	plugin := mp.NewMackerelPlugin(b)
	plugin.Run()

	return nil
}

func Do() int {
	err := parseArgs(os.Args[1:])
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	initLogger()

	err = run()
	if err != nil {
		return 1
	}
	return 0
}
