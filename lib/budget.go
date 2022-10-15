package lib

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/mackerelio/mackerel-client-go"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

const Name string = "mackerel-plugin-dns-lookup"
const Version string = "0.1.0"

var logger *zap.Logger
var opts options

type options struct {
	Origization      string  `short:"o" long:"org" description:"" required:"false"`
	Service          string  `short:"s" long:"service" description:"" required:"true"`
	Roles            string  `short:"r" long:"roles" description:"" required:"true"`
	Filter           string  `short:"f" long:"filter" description:"" required:"false"`
	Metrics          string  `short:"m" long:"metrics" description:"" required:"true"`
	ObjectiveLatency float64 `long:"objective-latency" description:"" required:"true"`
	ErrorBudgetSize  float64 `long:"errorbudget-size" description:"" default:"0.01" required:"false"`
	TimeWindow       int     `long:"timewindo" description:"" default:"30" required:"false"`
	Debug            bool    `long:"debug" description:"" required:"false"`
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
	for i := 1; i < 30; i++ {
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

func (m Mackerel) ObjecttiveEvaluation(res map[time.Time]float64, slo SLO) {
	var violationsCount float64
	for k, v := range res {
		if v > slo.ObjectivesLatency {
			logger.Debug("[DEBUG]", zap.String("ObjecttiveEvaluation", fmt.Sprintf("%s: %f", k, v)))
			violationsCount++
		}
	}
	fmt.Println(violationsCount)
	fmt.Println(100 - (violationsCount / calcTimeWindowMinutes(slo.TimeWindow)))
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
	cli.ObjecttiveEvaluation(calcAverage(res), slo)

	return nil
}

func Do() int {
	err := parseArgs(os.Args[1:])
	if err != nil {
		os.Exit(1)
	}
	initLogger()

	err = run()
	fmt.Println(err)
	if err != nil {
		return 1
	}
	return 0
}
