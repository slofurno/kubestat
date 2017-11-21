create table podstat (
	time timestamp with time zone, 
	dt bigint, 
	name text, 
	cpuacct_usage_d bigint, 
	throttled_time_d bigint, 
	total_rss bigint, 
	total_cache bigint, 
	total_mapped_file bigint,
	memory_limit bigint);
