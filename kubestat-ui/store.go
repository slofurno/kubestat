package main

import (
	"database/sql"
	"time"

	_ "github.com/lib/pq"
)

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

type Store struct {
	db *sql.DB
}

type PodStatQuery struct {
	start int
	end   int
	name  string
}

const podstatfields = `time, dt, name, cpuacct_usage_d, throttled_time_d, total_rss, total_cache, total_mapped_file, memory_limit`

func (s *Store) Get(q PodStatQuery) ([]*PodStat, error) {
	query := `select ` + podstatfields + ` from podstat where time >= now() - $1::INTERVAL and time < now() - $2::INTERVAL and name like $3`
	rows, err := s.db.Query(query, q.start, q.end, q.name+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ret := []*PodStat{}
	for rows.Next() {
		p := &PodStat{}
		if err := rows.Scan(
			&p.Time,
			&p.Dt,
			&p.Name,
			&p.Cpuacct_usage_d,
			&p.Throttled_time_d,
			&p.Total_rss,
			&p.Total_cache,
			&p.Total_mapped_file,
			&p.Hierarchical_memory_limit,
		); err != nil {
			return nil, err
		}

		ret = append(ret, p)
	}

	return ret, nil
}

func (s *Store) Put(xs []PodStat) error {
	query := `insert into podstat (time, dt, name, cpuacct_usage_d, throttled_time_d, total_rss, total_cache,
		total_mapped_file, memory_limit) values ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	for i := range xs {
		if _, err := s.db.Exec(query, xs[i].Time, xs[i].Dt, xs[i].Name, xs[i].Cpuacct_usage_d, xs[i].Throttled_time_d,
			xs[i].Total_rss, xs[i].Total_cache, xs[i].Total_mapped_file, xs[i].Hierarchical_memory_limit); err != nil {
			return err
		}
	}

	return nil
}

func NewPostgresStore(cs string) (*Store, error) {
	var db *sql.DB
	var err error

	if db, err = sql.Open("postgres", cs); err != nil {
		return nil, err
	}

	if err = db.Ping(); err != nil {
		return nil, err
	}

	return &Store{
		db: db,
	}, nil
}
