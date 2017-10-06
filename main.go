package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
	"time"
)

var rootDir string

func init() {
	flag.StringVar(&rootDir, "root", "/sys/fs/cgroup", "root directory")

	flag.Parse()
}

func main() {

	s := New(rootDir)

	for {
		s.Refresh()

		for id, pod := range s.pods {
			fmt.Printf("[%s] %#v\n", id, pod)
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
	cpuacct_usage    int64
	cpuacct_usage_d  int64
	nr_throttled     int64
	throttled_time   int64
	throttled_time_d int64
	total_rss        int64
	total_cache      int64
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
					pod = &PodStat{}
					s.pods[x.Name()] = pod
				}

				a := base + "/" + x.Name() + "/" + "cpuacct.usage"
				n, _ := ioutil.ReadFile(a)

				cu := strings.Split(string(n), "\n")[0]
				cpuacct_usage, _ := strconv.ParseInt(cu, 10, 64)
				if pod.cpuacct_usage > 0 {
					pod.cpuacct_usage_d = cpuacct_usage - pod.cpuacct_usage
				}
				pod.cpuacct_usage = cpuacct_usage

				b := base + "/" + x.Name() + "/" + "cpu.stat"
				n, _ = ioutil.ReadFile(b)

				lines := strings.Split(string(n), "\n")

				nr_throttled := strings.Split(lines[1], " ")[1]
				th := strings.Split(lines[2], " ")[1]

				pod.nr_throttled, _ = strconv.ParseInt(nr_throttled, 10, 64)
				throttled_time, _ := strconv.ParseInt(th, 10, 64)
				if pod.throttled_time > 0 {
					pod.throttled_time_d = throttled_time - pod.throttled_time
				}
				pod.throttled_time = throttled_time
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
						pod.total_rss, _ = strconv.ParseInt(next[1], 10, 64)

					case "total_cache":
						pod.total_cache, _ = strconv.ParseInt(next[1], 10, 64)
					}

				}
			}
		}

	}
}
