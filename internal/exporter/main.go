package exporter

import (
	"context"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	// "net/url"
	// "regexp"
	// "strconv"
	// "strings"
	// "time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	// "github.com/hashicorp/go-cleanhttp"
	"github.com/hashicorp/go-tfe"
)

const (
	namespace = "tfc"
)

var (
	up = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "up"),
		"Was the last query of Consul successful.",
		nil, nil,
	)
	workspaceAll = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "workspaces_all"),
		"How many workspaces in the organization.",
		[]string{"ID", "name", "version", "mode", "branch", "status"}, nil,
	)
	workspaceRunCount = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "workspace_runs"),
		"Workspace run count.",
		[]string{"ID", "name", "version", "mode", "branch", "status"}, nil,
	)
	workspaceFailureCount = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "workspace_failures"),
		"Workspace failure count.",
		[]string{"ID", "name", "version", "mode", "branch", "status"}, nil,
	)
	workspaceResourceCount = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "workspace_resources"),
		"How many resources in the workspace.",
		[]string{"ID", "name", "version", "mode", "branch", "status"}, nil,
	)

	clusterLeader = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "raft_leader"),
		"Does Raft cluster have a leader (according to this node).",
		nil, nil,
	)
	nodeCount = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "serf_lan_members"),
		"How many members are in the cluster.",
		nil, nil,
	)
	memberStatus = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "serf_lan_member_status"),
		"Status of member in the cluster. 1=Alive, 2=Leaving, 3=Left, 4=Failed.",
		[]string{"member"}, nil,
	)
	memberWanStatus = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "serf_wan_member_status"),
		"Status of member in the wan cluster. 1=Alive, 2=Leaving, 3=Left, 4=Failed.",
		[]string{"member", "dc"}, nil,
	)
	serviceCount = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "catalog_services"),
		"How many services are in the cluster.",
		nil, nil,
	)
	serviceTag = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "service_tag"),
		"Tags of a service.",
		[]string{"service_id", "node", "tag"}, nil,
	)
	serviceNodesHealthy = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "catalog_service_node_healthy"),
		"Is this service healthy on this node?",
		[]string{"service_id", "node", "service_name"}, nil,
	)
	nodeChecks = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "health_node_status"),
		"Status of health checks associated with a node.",
		[]string{"check", "node", "status"}, nil,
	)
	serviceChecks = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "health_service_status"),
		"Status of health checks associated with a service.",
		[]string{"check", "node", "service_id", "service_name", "status"}, nil,
	)
	serviceCheckNames = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "service_checks"),
		"Link the service id and check name if available.",
		[]string{"service_id", "service_name", "check_id", "check_name", "node"}, nil,
	)
	keyValues = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "catalog_kv"),
		"The values for selected keys in Consul's key/value catalog. Keys with non-numeric values are omitted.",
		[]string{"key"}, nil,
	)
)

// Exporter collects Terraform cloud stats for the given organization and exports them using
// the prometheus metrics package.
type Exporter struct {
	client           *tfe.Client
	organization     string
	healthSummary    bool
	logger           log.Logger
	requestLimitChan chan struct{}
}

// New returns an initialized Exporter.
func New(organization string, token string, healthSummary bool, logger log.Logger) (*Exporter, error) {
	config := &tfe.Config{
		Token: token,
		RetryLogHook: func(attempNum int, resp *http.Response) {
			level.Debug(logger).Log("msg", fmt.Sprintf("Retry ", attempNum, "times"), "resp", resp)
		},
	}
	client, err := tfe.NewClient(config)
	if err != nil {
		level.Error(logger).Log("msg", "Can't create Terraform cloud client", "err", err)
		return nil, err
	}
	org, err := client.Organizations.Read(context.Background(), organization)
	if err != nil {
		level.Error(logger).Log("msg", "Can't create get organization", "err", err)
		return nil, err
	}
	level.Debug(logger).Log("msg", fmt.Sprintf("Getting organization information %s with ID %s", org.Name, org.ExternalID))

	var requestLimitChan chan struct{}
	// if opts.RequestLimit > 0 {
	// 	requestLimitChan = make(chan struct{}, opts.RequestLimit)
	// }

	// Init our exporter.
	return &Exporter{
		client:           client,
		organization:     organization,
		healthSummary:    healthSummary,
		logger:           logger,
		requestLimitChan: requestLimitChan,
	}, nil
}

// Describe describes all the metrics ever exported by the Terraform Cloud exporter. It
// implements prometheus.Collector.
func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- up
	ch <- workspaceRunCount
	ch <- workspaceFailureCount
	ch <- workspaceResourceCount
	// ch <- clusterLeader
	// ch <- nodeCount
	// ch <- memberStatus
	// ch <- memberWanStatus
	// ch <- serviceCount
	// ch <- serviceNodesHealthy
	// ch <- nodeChecks
	// ch <- serviceChecks
	// ch <- keyValues
	// ch <- serviceTag
	// ch <- serviceCheckNames
}

// Collect fetches the stats from configured Terraform Cloud location and delivers them
// as Prometheus metrics. It implements prometheus.Collector.
func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	ok := e.collectWorkspaceMetric(ch)
	// ok = e.collectLeaderMetric(ch) && ok
	// ok = e.collectNodesMetric(ch) && ok
	// ok = e.collectMembersMetric(ch) && ok
	// ok = e.collectMembersWanMetric(ch) && ok
	// ok = e.collectServicesMetric(ch) && ok
	// ok = e.collectHealthStateMetric(ch) && ok
	// ok = e.collectKeyValues(ch) && ok

	if ok {
		ch <- prometheus.MustNewConstMetric(
			up, prometheus.GaugeValue, 1.0,
		)
	} else {
		ch <- prometheus.MustNewConstMetric(
			up, prometheus.GaugeValue, 0.0,
		)
	}
}

func (e *Exporter) collectWorkspaceMetric(ch chan<- prometheus.Metric) bool {
	// peers, err := e.client.Status().Peers()
	workspaces, err := e.getAllWorkspaces()
	if err != nil {
		level.Error(e.logger).Log("msg", "Can't query workspaces", "err", err)
		return false
	}
	for _, workspace := range workspaces {
		latestStatus := "unknow"
		// run, err := e.getLastRun(workspace)
		// if err != nil {
		// 	level.Error(e.logger).Log("msg", "Can't query runs", "err", err)
		// } else {
		// 	latestStatus = string(run.Status)
		// 	level.Debug(e.logger).Log("msg", latestStatus, "status", run.Status)
		// }

		ch <- prometheus.MustNewConstMetric(
			workspaceAll,
			prometheus.GaugeValue,
			1.0,
			workspace.ID,
			workspace.Name,
			workspace.TerraformVersion,
			workspace.ExecutionMode,
			workspace.VCSRepo.Branch,
			latestStatus,
		)
		ch <- prometheus.MustNewConstMetric(
			workspaceRunCount,
			prometheus.GaugeValue,
			float64(workspace.RunsCount),
			workspace.ID,
			workspace.Name,
			workspace.TerraformVersion,
			workspace.ExecutionMode,
			workspace.VCSRepo.Branch,
			latestStatus,
		)
		ch <- prometheus.MustNewConstMetric(
			workspaceFailureCount,
			prometheus.GaugeValue,
			float64(workspace.RunFailures),
			workspace.ID,
			workspace.Name,
			workspace.TerraformVersion,
			workspace.ExecutionMode,
			workspace.VCSRepo.Branch,
			latestStatus,
		)
		ch <- prometheus.MustNewConstMetric(
			workspaceResourceCount,
			prometheus.GaugeValue,
			float64(workspace.ResourceCount),
			workspace.ID,
			workspace.Name,
			workspace.TerraformVersion,
			workspace.ExecutionMode,
			workspace.VCSRepo.Branch,
			latestStatus,
		)
	}
	return true
}

// func (e *Exporter) collectLeaderMetric(ch chan<- prometheus.Metric) bool {
//     leader, err := e.client.Status().Leader()
//     if err != nil {
//         level.Error(e.logger).Log("msg", "Can't query consul", "err", err)
//         return false
//     }
//     if len(leader) == 0 {
//         ch <- prometheus.MustNewConstMetric(
//             clusterLeader, prometheus.GaugeValue, 0,
//         )
//     } else {
//         ch <- prometheus.MustNewConstMetric(
//             clusterLeader, prometheus.GaugeValue, 1,
//         )
//     }
//     return true
// }

// func (e *Exporter) collectNodesMetric(ch chan<- prometheus.Metric) bool {
//     nodes, _, err := e.client.Catalog().Nodes(&e.queryOptions)
//     if err != nil {
//         level.Error(e.logger).Log("msg", "Failed to query catalog for nodes", "err", err)
//         return false
//     }
//     ch <- prometheus.MustNewConstMetric(
//         nodeCount, prometheus.GaugeValue, float64(len(nodes)),
//     )
//     return true
// }

// func (e *Exporter) collectMembersMetric(ch chan<- prometheus.Metric) bool {
//     members, err := e.client.Agent().Members(false)
//     if err != nil {
//         level.Error(e.logger).Log("msg", "Failed to query member status", "err", err)
//         return false
//     }
//     for _, entry := range members {
//         ch <- prometheus.MustNewConstMetric(
//             memberStatus, prometheus.GaugeValue, float64(entry.Status), entry.Name,
//         )
//     }
//     return true
// }

// func (e *Exporter) collectMembersWanMetric(ch chan<- prometheus.Metric) bool {
//     members, err := e.client.Agent().Members(true)
//     if err != nil {
//         level.Error(e.logger).Log("msg", "Failed to query wan member status", "err", err)
//         return false
//     }
//     for _, entry := range members {
//         ch <- prometheus.MustNewConstMetric(
//             memberWanStatus, prometheus.GaugeValue, float64(entry.Status), entry.Name, entry.Tags["dc"],
//         )
//     }
//     return true
// }

// func (e *Exporter) collectServicesMetric(ch chan<- prometheus.Metric) bool {
//     serviceNames, _, err := e.client.Catalog().Services(&e.queryOptions)
//     if err != nil {
//         level.Error(e.logger).Log("msg", "Failed to query for services", "err", err)
//         return false
//     }
//     ch <- prometheus.MustNewConstMetric(
//         serviceCount, prometheus.GaugeValue, float64(len(serviceNames)),
//     )
//     if e.healthSummary {
//         if ok := e.collectHealthSummary(ch, serviceNames); !ok {
//             return false
//         }
//     }
//     return true
// }

// func (e *Exporter) collectHealthStateMetric(ch chan<- prometheus.Metric) bool {
//     checks, _, err := e.client.Health().State("any", &e.queryOptions)
//     if err != nil {
//         level.Error(e.logger).Log("msg", "Failed to query service health", "err", err)
//         return false
//     }
//     for _, hc := range checks {
//         var passing, warning, critical, maintenance float64

//         switch hc.Status {
//         case consul_api.HealthPassing:
//             passing = 1
//         case consul_api.HealthWarning:
//             warning = 1
//         case consul_api.HealthCritical:
//             critical = 1
//         case consul_api.HealthMaint:
//             maintenance = 1
//         }

//         if hc.ServiceID == "" {
//             ch <- prometheus.MustNewConstMetric(
//                 nodeChecks, prometheus.GaugeValue, passing, hc.CheckID, hc.Node, consul_api.HealthPassing,
//             )
//             ch <- prometheus.MustNewConstMetric(
//                 nodeChecks, prometheus.GaugeValue, warning, hc.CheckID, hc.Node, consul_api.HealthWarning,
//             )
//             ch <- prometheus.MustNewConstMetric(
//                 nodeChecks, prometheus.GaugeValue, critical, hc.CheckID, hc.Node, consul_api.HealthCritical,
//             )
//             ch <- prometheus.MustNewConstMetric(
//                 nodeChecks, prometheus.GaugeValue, maintenance, hc.CheckID, hc.Node, consul_api.HealthMaint,
//             )
//         } else {
//             ch <- prometheus.MustNewConstMetric(
//                 serviceChecks, prometheus.GaugeValue, passing, hc.CheckID, hc.Node, hc.ServiceID, hc.ServiceName, consul_api.HealthPassing,
//             )
//             ch <- prometheus.MustNewConstMetric(
//                 serviceChecks, prometheus.GaugeValue, warning, hc.CheckID, hc.Node, hc.ServiceID, hc.ServiceName, consul_api.HealthWarning,
//             )
//             ch <- prometheus.MustNewConstMetric(
//                 serviceChecks, prometheus.GaugeValue, critical, hc.CheckID, hc.Node, hc.ServiceID, hc.ServiceName, consul_api.HealthCritical,
//             )
//             ch <- prometheus.MustNewConstMetric(
//                 serviceChecks, prometheus.GaugeValue, maintenance, hc.CheckID, hc.Node, hc.ServiceID, hc.ServiceName, consul_api.HealthMaint,
//             )
//             ch <- prometheus.MustNewConstMetric(
//                 serviceCheckNames, prometheus.GaugeValue, 1, hc.ServiceID, hc.ServiceName, hc.CheckID, hc.Name, hc.Node,
//             )
//         }
//     }
//     return true
// }

// // collectHealthSummary collects health information about every node+service
// // combination. It will cause one lookup query per service.
// func (e *Exporter) collectHealthSummary(ch chan<- prometheus.Metric, serviceNames map[string][]string) bool {
//     ok := make(chan bool)

//     for s := range serviceNames {
//         go func(s string) {
//             if e.requestLimitChan != nil {
//                 e.requestLimitChan <- struct{}{}
//                 defer func() {
//                     <-e.requestLimitChan
//                 }()
//             }
//             ok <- e.collectOneHealthSummary(ch, s)
//         }(s)
//     }

//     allOK := true
//     for range serviceNames {
//         allOK = <-ok && allOK
//     }
//     close(ok)

//     return allOK
// }

// func (e *Exporter) collectOneHealthSummary(ch chan<- prometheus.Metric, serviceName string) bool {
//     // See https://github.com/hashicorp/consul/issues/1096.
//     if strings.HasPrefix(serviceName, "/") {
//         level.Warn(e.logger).Log("msg", "Skipping service because it starts with a slash", "service_name", serviceName)
//         return true
//     }
//     level.Debug(e.logger).Log("msg", "Fetching health summary", "serviceName", serviceName)

//     service, _, err := e.client.Health().Service(serviceName, "", false, &e.queryOptions)
//     if err != nil {
//         level.Error(e.logger).Log("msg", "Failed to query service health", "err", err)
//         return false
//     }

//     for _, entry := range service {
//         // We have a Node, a Service, and one or more Checks. Our
//         // service-node combo is passing if all checks have a `status`
//         // of "passing."
//         passing := 1.
//         for _, hc := range entry.Checks {
//             if hc.Status != consul_api.HealthPassing {
//                 passing = 0
//                 break
//             }
//         }
//         ch <- prometheus.MustNewConstMetric(
//             serviceNodesHealthy, prometheus.GaugeValue, passing, entry.Service.ID, entry.Node.Node, entry.Service.Service,
//         )
//         tags := make(map[string]struct{})
//         for _, tag := range entry.Service.Tags {
//             if _, ok := tags[tag]; ok {
//                 continue
//             }
//             ch <- prometheus.MustNewConstMetric(serviceTag, prometheus.GaugeValue, 1, entry.Service.ID, entry.Node.Node, tag)
//             tags[tag] = struct{}{}
//         }
//     }
//     return true
// }

// func (e *Exporter) collectKeyValues(ch chan<- prometheus.Metric) bool {
//     if e.kvPrefix == "" {
//         return true
//     }

//     kv := e.client.KV()
//     pairs, _, err := kv.List(e.kvPrefix, &e.queryOptions)
//     if err != nil {
//         level.Error(e.logger).Log("msg", "Error fetching key/values", "err", err)
//         return false
//     }

//     for _, pair := range pairs {
//         if e.kvFilter.MatchString(pair.Key) {
//             val, err := strconv.ParseFloat(string(pair.Value), 64)
//             if err == nil {
//                 ch <- prometheus.MustNewConstMetric(
//                     keyValues, prometheus.GaugeValue, val, pair.Key,
//                 )
//             }
//         }
//     }
//     return true
// }
