package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var rootDir string

type NameMap struct {
	pods   map[string]string
	client *kubernetes.Clientset
}

var nameMap *NameMap

func (s *NameMap) Refresh() {
	pods, err := s.client.CoreV1().Pods("").List(metav1.ListOptions{})
	if err != nil {
		return
	}

	s.pods = map[string]string{}
	for i := range pods.Items {
		uid, name := pods.Items[i].GetUID(), pods.Items[i].GetName()
		s.pods[string(uid)] = name
	}
}

func (s *NameMap) Lookup(uid string) (string, bool) {
	if name, ok := s.pods[uid]; ok {
		return name, true
	}
	return "", false
}

func init() {
	flag.StringVar(&rootDir, "root", "/sys/fs/cgroup", "root directory")

	flag.Parse()
}

func main() {

	endpoint := os.Getenv("DRAIN_ENDPOINT")

	client := http.Client{Timeout: 5 * time.Second}

	s := New(rootDir)

	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}
	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	nameMap = &NameMap{
		client: clientset,
		pods:   map[string]string{},
	}

	for i := 0; ; i = (i & 15) + 1 {
		if i == 0 {
			nameMap.Refresh()
		}
		s.Refresh()

		pods := []*PodStat{}
		for _, pod := range s.pods {
			pods = append(pods, pod)
		}

		b, _ := json.Marshal(pods)
		req, _ := http.NewRequest("POST", endpoint, bytes.NewReader(b))
		res, err := client.Do(req)
		if err != nil {
			fmt.Println(err)
		} else {
			ioutil.ReadAll(res.Body)
			res.Body.Close()
		}

		time.Sleep(5 * time.Second)
	}

}

type Stats struct {
	root string
	pods map[string]*PodStat
}

func New(root string) *Stats {
	return &Stats{
		root: root,
		pods: map[string]*PodStat{},
	}
}

type PodStat struct {
	Id   string
	Name string
	//nanoseconds
	Cpuacct_usage             int64
	Cpuacct_usage_d           int64
	Nr_throttled              int64
	Throttled_time            int64
	Throttled_time_d          int64
	Total_rss                 int64
	Total_cache               int64
	Total_mapped_file         int64
	Hierarchical_memory_limit int64

	//microseconds
	Cpu_cfs_quota_us  int64
	Cpu_cfs_period_us int64

	Time time.Time
	Dt   time.Duration

	named bool
}

func matchName(n string) bool {
	return strings.HasPrefix(n, "pod") && strings.Contains(n, "-")
}

func (s *Stats) Refresh() {
	for _, b := range []string{"burstable", "besteffort"} {
		base := s.root + "/cpu/kubepods/" + b
		xs, _ := ioutil.ReadDir(base)

		for _, x := range xs {
			if x.IsDir() && matchName(x.Name()) {

				var pod *PodStat
				var ok bool
				if pod, ok = s.pods[x.Name()]; !ok {
					pod = &PodStat{Id: x.Name()[3:]}
					s.pods[x.Name()] = pod
				}

				pod.Dt = time.Since(pod.Time)
				pod.Time = time.Now()

				if !pod.named {
					if name, ok := nameMap.Lookup(pod.Id); ok {
						pod.Name = name
						pod.named = true
					}
				}

				a := base + "/" + x.Name() + "/" + "cpuacct.usage"
				n, _ := ioutil.ReadFile(a)

				cu := strings.Split(string(n), "\n")[0]
				cpuacct_usage, _ := strconv.ParseInt(cu, 10, 64)
				if pod.Cpuacct_usage > 0 {
					pod.Cpuacct_usage_d = cpuacct_usage - pod.Cpuacct_usage
				}
				pod.Cpuacct_usage = cpuacct_usage

				b := base + "/" + x.Name() + "/" + "cpu.stat"
				n, _ = ioutil.ReadFile(b)

				lines := strings.Split(string(n), "\n")

				nr_throttled := strings.Split(lines[1], " ")[1]
				th := strings.Split(lines[2], " ")[1]

				pod.Nr_throttled, _ = strconv.ParseInt(nr_throttled, 10, 64)
				throttled_time, _ := strconv.ParseInt(th, 10, 64)
				if pod.Throttled_time > 0 {
					pod.Throttled_time_d = throttled_time - pod.Throttled_time
				}
				pod.Throttled_time = throttled_time

				b = base + "/" + x.Name() + "/" + "cpu.cfs_period_us"
				n, _ = ioutil.ReadFile(b)

				period := strings.Split(string(n), "\n")[0]
				pod.Cpu_cfs_period_us, _ = strconv.ParseInt(period, 10, 64)

				b = base + "/" + x.Name() + "/" + "cpu.cfs_quota_us"
				n, _ = ioutil.ReadFile(b)

				quota := strings.Split(string(n), "\n")[0]
				pod.Cpu_cfs_quota_us, _ = strconv.ParseInt(quota, 10, 64)
			}
		}

		base = s.root + "/memory/kubepods/" + b
		xs, _ = ioutil.ReadDir(base)

		for _, x := range xs {
			if x.IsDir() && matchName(x.Name()) {

				var pod *PodStat
				var ok bool
				if pod, ok = s.pods[x.Name()]; !ok {
					pod = &PodStat{}
					s.pods[x.Name()] = pod
				}

				a := base + "/" + x.Name() + "/" + "memory.stat"
				n, _ := ioutil.ReadFile(a)

				lines := strings.Split(string(n), "\n")

				for i := range lines {
					next := strings.Split(lines[i], " ")
					switch next[0] {
					case "total_rss":
						pod.Total_rss, _ = strconv.ParseInt(next[1], 10, 64)

					case "total_cache":
						pod.Total_cache, _ = strconv.ParseInt(next[1], 10, 64)

					case "total_mapped_file":
						pod.Total_mapped_file, _ = strconv.ParseInt(next[1], 10, 64)

					case "hierarchical_memory_limit":
						pod.Hierarchical_memory_limit, _ = strconv.ParseInt(next[1], 10, 64)
					}

				}
			}
		}

	}
}
