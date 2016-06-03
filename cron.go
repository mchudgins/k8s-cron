package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	flag "github.com/spf13/pflag"

	"github.com/mchudgins/cron"
	"github.com/mchudgins/k8s-helpers/pkg/clientConfig"
	election "github.com/mchudgins/k8s-helpers/pkg/leader"
	"k8s.io/kubernetes/pkg/client/restclient"
	client "k8s.io/kubernetes/pkg/client/unversioned"
)

const (
	podNamespace string = "POD_NAMESPACE"
)

var (
	flags = flag.NewFlagSet(
		`cron`,
		flag.ExitOnError)
	name      = flags.String("election", "cron", "The name of the election")
	id        = flags.String("id", "", "The id of this participant")
	namespace = flags.String("election-namespace", "", "The Kubernetes namespace for this election")
	ttl       = flags.Duration("ttl", 10*time.Second, "The TTL for this election")
	inCluster = flags.Bool("use-cluster-credentials", false, "Should this request use cluster credentials?")
	addr      = flags.String("http", "", "If non-empty, stand up a simple webserver that reports the leader state")

	fLeader = false
	leader  = &LeaderData{}
)

// LeaderData represents information about the current leader
type LeaderData struct {
	Name string `json:"name"`
}

func main() {
	flags.Parse(os.Args)
	validateFlags()

	// create the cron struct and, via CronTab(), the
	// events we'll fire and the intervals for each
	cron := cron.New()
	CronTab(cron)

	// to ensure High Availability, we assume that there
	// are multiple cron agents.  Only one of these may
	// actually fire events.  The next bit of code
	// use the k8s "leader election" api to select the agent
	// which will run cron.Start()
	fn := func(str string) {
		leader.Name = str
		fmt.Printf("%s is the leader\n", leader.Name)
		if strings.Compare(leader.Name, *id) == 0 {
			if !fLeader {
				cron.Start()
			}
			fLeader = true
		} else {
			if fLeader {
				cron.Stop()
			}
			fLeader = false
		}
	}

	kubeClient, err := makeClient()
	if err != nil {
		log.Fatalf("error connecting to the client: %v", err)
	}

	e, err := election.NewElection(*name, *id, *namespace, *ttl, fn, kubeClient)
	if err != nil {
		log.Fatalf("failed to create election: %v", err)
	}
	go election.RunElection(e)

	// if they want, run a webserver to show current status
	if len(*addr) > 0 {
		http.HandleFunc("/", webHandler)

		SetupMetrics()

		http.ListenAndServe(*addr, nil)
	} else {
		select {}
	}
}

// make sure we have the data we need
func validateFlags() {

	if len(*id) == 0 {

		if *inCluster {
			var err error
			if *id, err = os.Hostname(); err != nil || len(*id) == 0 {
				log.Fatal(err)
			}
		} else {
			log.Fatal("id flag is required when running outside the cluster")
		}

	}

	if len(*name) == 0 {
		filename := os.Args[0]
		*name = path.Base(filename)[:len(path.Ext(filename))-2]
	}

	if len(*namespace) == 0 {
		if *inCluster {
			*namespace = os.Getenv(podNamespace)
			if len(*namespace) == 0 {
				log.Fatalf("unable to obtain current namespace from Environment variable %s", podNamespace)
			}
		} else {
			k8scfg, err := clientConfig.KubeConfig()
			if err != nil {
				log.Fatal("unable to retrieve KubeConfig")
			}
			ctx, err := k8scfg.ActiveContext()
			if err != nil {
				log.Fatal("unable to obtain active contact of kubeConfig")
			}
			*namespace = ctx.Context.Namespace
		}
	}
}

//
func makeClient() (*client.Client, error) {
	var cfg *restclient.Config
	var err error

	if *inCluster {
		if cfg, err = restclient.InClusterConfig(); err != nil {
			return nil, err
		}
	} else {
		if cfg, err = clientConfig.NewConfig(); err != nil {
			return nil, err
		}
	}

	return client.New(cfg)
}

func webHandler(res http.ResponseWriter, req *http.Request) {
	data, err := json.Marshal(leader)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		res.Write([]byte(err.Error()))
		return
	}
	res.WriteHeader(http.StatusOK)
	res.Write(data)
}
