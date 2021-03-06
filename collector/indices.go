package collector

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/client_golang/prometheus"

	awsauth "github.com/smartystreets/go-aws-auth"
)

var (
	defaultIndexLabels      = []string{"index"}
	defaultIndexLabelValues = func(indexName string) []string {
		return []string{indexName}
	}
)

type indexMetric struct {
	Type   prometheus.ValueType
	Desc   *prometheus.Desc
	Value  func(indexStats IndexStatsIndexResponse) float64
	Labels func(indexName string) []string
}

type Indices struct {
	logger log.Logger
	client *http.Client
	url    *url.URL

	up                prometheus.Gauge
	totalScrapes      prometheus.Counter
	jsonParseFailures prometheus.Counter

	indexMetrics []*indexMetric
}

func NewIndices(logger log.Logger, client *http.Client, url *url.URL) *Indices {
	return &Indices{
		logger: logger,
		client: client,
		url:    url,

		up: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: prometheus.BuildFQName(namespace, "index_stats", "up"),
			Help: "Was the last scrape of the ElasticSearch index endpoint successful.",
		}),
		totalScrapes: prometheus.NewCounter(prometheus.CounterOpts{
			Name: prometheus.BuildFQName(namespace, "index_stats", "total_scrapes"),
			Help: "Current total ElasticSearch index scrapes.",
		}),
		jsonParseFailures: prometheus.NewCounter(prometheus.CounterOpts{
			Name: prometheus.BuildFQName(namespace, "index_stats", "json_parse_failures"),
			Help: "Number of errors while parsing JSON.",
		}),

		indexMetrics: []*indexMetric{
			{
				Type: prometheus.GaugeValue,
				Desc: prometheus.NewDesc(
					prometheus.BuildFQName(namespace, "indices", "docs_primary"),
					"Count of documents with only primary shards",
					defaultIndexLabels, nil,
				),
				Value: func(indexStats IndexStatsIndexResponse) float64 {
					return float64(indexStats.Primaries.Docs.Count)
				},
				Labels: defaultIndexLabelValues,
			},
			{
				Type: prometheus.GaugeValue,
				Desc: prometheus.NewDesc(
					prometheus.BuildFQName(namespace, "indices", "store_size_bytes_primary"),
					"Current total size of stored index data in bytes with only primary shards on all nodes",
					defaultIndexLabels, nil,
				),
				Value: func(indexStats IndexStatsIndexResponse) float64 {
					return float64(indexStats.Primaries.Store.SizeInBytes)
				},
				Labels: defaultIndexLabelValues,
			},
			{
				Type: prometheus.GaugeValue,
				Desc: prometheus.NewDesc(
					prometheus.BuildFQName(namespace, "indices", "store_size_bytes_total"),
					"Current total size of stored index data in bytes with all shards on all nodes",
					defaultIndexLabels, nil,
				),
				Value: func(indexStats IndexStatsIndexResponse) float64 {
					return float64(indexStats.Total.Store.SizeInBytes)
				},
				Labels: defaultIndexLabelValues,
			},
			{
				Type: prometheus.GaugeValue,
				Desc: prometheus.NewDesc(
					prometheus.BuildFQName(namespace, "indices", "segment_count_primary"),
					"Current number of segments with only primary shards on all nodes",
					defaultIndexLabels, nil,
				),
				Value: func(indexStats IndexStatsIndexResponse) float64 {
					return float64(indexStats.Primaries.Segments.Count)
				},
				Labels: defaultIndexLabelValues,
			},
			{
				Type: prometheus.GaugeValue,
				Desc: prometheus.NewDesc(
					prometheus.BuildFQName(namespace, "indices", "segment_count_total"),
					"Current number of segments with all shards on all nodes",
					defaultIndexLabels, nil,
				),
				Value: func(indexStats IndexStatsIndexResponse) float64 {
					return float64(indexStats.Total.Segments.Count)
				},
				Labels: defaultIndexLabelValues,
			},
			{
				Type: prometheus.GaugeValue,
				Desc: prometheus.NewDesc(
					prometheus.BuildFQName(namespace, "indices", "segment_memory_bytes_primary"),
					"Current size of segments with only primary shards on all nodes in bytes",
					defaultIndexLabels, nil,
				),
				Value: func(indexStats IndexStatsIndexResponse) float64 {
					return float64(indexStats.Primaries.Segments.MemoryInBytes)
				},
				Labels: defaultIndexLabelValues,
			},
			{
				Type: prometheus.GaugeValue,
				Desc: prometheus.NewDesc(
					prometheus.BuildFQName(namespace, "indices", "segment_memory_bytes_total"),
					"Current size of segments with all shards on all nodes in bytes",
					defaultIndexLabels, nil,
				),
				Value: func(indexStats IndexStatsIndexResponse) float64 {
					return float64(indexStats.Total.Segments.MemoryInBytes)
				},
				Labels: defaultIndexLabelValues,
			},
			{
				Type: prometheus.GaugeValue,
				Desc: prometheus.NewDesc(
					prometheus.BuildFQName(namespace, "indices", "segment_terms_memory_primary"),
					"Current size of terms with only primary shards on all nodes in bytes",
					defaultIndexLabels, nil,
				),
				Value: func(indexStats IndexStatsIndexResponse) float64 {
					return float64(indexStats.Primaries.Segments.TermsMemoryInBytes)
				},
				Labels: defaultIndexLabelValues,
			},
			{
				Type: prometheus.GaugeValue,
				Desc: prometheus.NewDesc(
					prometheus.BuildFQName(namespace, "indices", "segment_terms_memory_total"),
					"Current number of terms with all shards on all nodes in bytes",
					defaultIndexLabels, nil,
				),
				Value: func(indexStats IndexStatsIndexResponse) float64 {
					return float64(indexStats.Total.Segments.TermsMemoryInBytes)
				},
				Labels: defaultIndexLabelValues,
			},
			{
				Type: prometheus.GaugeValue,
				Desc: prometheus.NewDesc(
					prometheus.BuildFQName(namespace, "indices", "segment_fields_memory_bytes_primary"),
					"Current size of fields with only primary shards on all nodes in bytes",
					defaultIndexLabels, nil,
				),
				Value: func(indexStats IndexStatsIndexResponse) float64 {
					return float64(indexStats.Primaries.Segments.StoredFieldsMemoryInBytes)
				},
				Labels: defaultIndexLabelValues,
			},
			{
				Type: prometheus.GaugeValue,
				Desc: prometheus.NewDesc(
					prometheus.BuildFQName(namespace, "indices", "segment_fields_memory_bytes_total"),
					"Current size of fields with all shards on all nodes in bytes",
					defaultIndexLabels, nil,
				),
				Value: func(indexStats IndexStatsIndexResponse) float64 {
					return float64(indexStats.Total.Segments.StoredFieldsMemoryInBytes)
				},
				Labels: defaultIndexLabelValues,
			},
			{
				Type: prometheus.GaugeValue,
				Desc: prometheus.NewDesc(
					prometheus.BuildFQName(namespace, "indices", "segment_norms_memory_bytes_primary"),
					"Current size of norms with only primary shards on all nodes in bytes",
					defaultIndexLabels, nil,
				),
				Value: func(indexStats IndexStatsIndexResponse) float64 {
					return float64(indexStats.Primaries.Segments.NormsMemoryInBytes)
				},
				Labels: defaultIndexLabelValues,
			},
			{
				Type: prometheus.GaugeValue,
				Desc: prometheus.NewDesc(
					prometheus.BuildFQName(namespace, "indices", "segment_norms_memory_bytes_total"),
					"Current size of norms with all shards on all nodes in bytes",
					defaultIndexLabels, nil,
				),
				Value: func(indexStats IndexStatsIndexResponse) float64 {
					return float64(indexStats.Total.Segments.NormsMemoryInBytes)
				},
				Labels: defaultIndexLabelValues,
			},
			{
				Type: prometheus.GaugeValue,
				Desc: prometheus.NewDesc(
					prometheus.BuildFQName(namespace, "indices", "segment_points_memory_bytes_primary"),
					"Current size of points with only primary shards on all nodes in bytes",
					defaultIndexLabels, nil,
				),
				Value: func(indexStats IndexStatsIndexResponse) float64 {
					return float64(indexStats.Primaries.Segments.PointsMemoryInBytes)
				},
				Labels: defaultIndexLabelValues,
			},
			{
				Type: prometheus.GaugeValue,
				Desc: prometheus.NewDesc(
					prometheus.BuildFQName(namespace, "indices", "segment_points_memory_bytes_total"),
					"Current size of points with all shards on all nodes in bytes",
					defaultIndexLabels, nil,
				),
				Value: func(indexStats IndexStatsIndexResponse) float64 {
					return float64(indexStats.Total.Segments.PointsMemoryInBytes)
				},
				Labels: defaultIndexLabelValues,
			},
			{
				Type: prometheus.GaugeValue,
				Desc: prometheus.NewDesc(
					prometheus.BuildFQName(namespace, "indices", "segment_doc_values_memory_bytes_primary"),
					"Current size of doc values with only primary shards on all nodes in bytes",
					defaultIndexLabels, nil,
				),
				Value: func(indexStats IndexStatsIndexResponse) float64 {
					return float64(indexStats.Primaries.Segments.DocValuesMemoryInBytes)
				},
				Labels: defaultIndexLabelValues,
			},
			{
				Type: prometheus.GaugeValue,
				Desc: prometheus.NewDesc(
					prometheus.BuildFQName(namespace, "indices", "segment_doc_values_memory_bytes_total"),
					"Current size of doc values with all shards on all nodes in bytes",
					defaultIndexLabels, nil,
				),
				Value: func(indexStats IndexStatsIndexResponse) float64 {
					return float64(indexStats.Total.Segments.DocValuesMemoryInBytes)
				},
				Labels: defaultIndexLabelValues,
			},
			{
				Type: prometheus.GaugeValue,
				Desc: prometheus.NewDesc(
					prometheus.BuildFQName(namespace, "indices", "segment_index_writer_memory_bytes_primary"),
					"Current size of index writer with only primary shards on all nodes in bytes",
					defaultIndexLabels, nil,
				),
				Value: func(indexStats IndexStatsIndexResponse) float64 {
					return float64(indexStats.Primaries.Segments.IndexWriterMemoryInBytes)
				},
				Labels: defaultIndexLabelValues,
			},
			{
				Type: prometheus.GaugeValue,
				Desc: prometheus.NewDesc(
					prometheus.BuildFQName(namespace, "indices", "segment_index_writer_memory_bytes_total"),
					"Current size of index writer with all shards on all nodes in bytes",
					defaultIndexLabels, nil,
				),
				Value: func(indexStats IndexStatsIndexResponse) float64 {
					return float64(indexStats.Total.Segments.IndexWriterMemoryInBytes)
				},
				Labels: defaultIndexLabelValues,
			},
			{
				Type: prometheus.GaugeValue,
				Desc: prometheus.NewDesc(
					prometheus.BuildFQName(namespace, "indices", "segment_version_map_memory_bytes_primary"),
					"Current size of version map with only primary shards on all nodes in bytes",
					defaultIndexLabels, nil,
				),
				Value: func(indexStats IndexStatsIndexResponse) float64 {
					return float64(indexStats.Primaries.Segments.VersionMapMemoryInBytes)
				},
				Labels: defaultIndexLabelValues,
			},
			{
				Type: prometheus.GaugeValue,
				Desc: prometheus.NewDesc(
					prometheus.BuildFQName(namespace, "indices", "segment_version_map_memory_bytes_total"),
					"Current size of version map with all shards on all nodes in bytes",
					defaultIndexLabels, nil,
				),
				Value: func(indexStats IndexStatsIndexResponse) float64 {
					return float64(indexStats.Total.Segments.VersionMapMemoryInBytes)
				},
				Labels: defaultIndexLabelValues,
			},
			{
				Type: prometheus.GaugeValue,
				Desc: prometheus.NewDesc(
					prometheus.BuildFQName(namespace, "indices", "segment_fixed_bit_set_memory_bytes_primary"),
					"Current size of fixed bit with only primary shards on all nodes in bytes",
					defaultIndexLabels, nil,
				),
				Value: func(indexStats IndexStatsIndexResponse) float64 {
					return float64(indexStats.Primaries.Segments.FixedBitSetMemoryInBytes)
				},
				Labels: defaultIndexLabelValues,
			},
			{
				Type: prometheus.GaugeValue,
				Desc: prometheus.NewDesc(
					prometheus.BuildFQName(namespace, "indices", "segment_fixed_bit_set_memory_bytes_total"),
					"Current size of fixed bit with all shards on all nodes in bytes",
					defaultIndexLabels, nil,
				),
				Value: func(indexStats IndexStatsIndexResponse) float64 {
					return float64(indexStats.Total.Segments.FixedBitSetMemoryInBytes)
				},
				Labels: defaultIndexLabelValues,
			},
			{
				Type: prometheus.GaugeValue,
				Desc: prometheus.NewDesc(
					prometheus.BuildFQName(namespace, "indices", "completion_bytes_primary"),
					"Current size of completion with only primary shards on all nodes in bytes",
					defaultIndexLabels, nil,
				),
				Value: func(indexStats IndexStatsIndexResponse) float64 {
					return float64(indexStats.Primaries.Completion.SizeInBytes)
				},
				Labels: defaultIndexLabelValues,
			},
			{
				Type: prometheus.GaugeValue,
				Desc: prometheus.NewDesc(
					prometheus.BuildFQName(namespace, "indices", "completion_bytes_total"),
					"Current size of completion with all shards on all nodes in bytes",
					defaultIndexLabels, nil,
				),
				Value: func(indexStats IndexStatsIndexResponse) float64 {
					return float64(indexStats.Total.Completion.SizeInBytes)
				},
				Labels: defaultIndexLabelValues,
			},
		},
	}
}

func (i *Indices) Describe(ch chan<- *prometheus.Desc) {
	for _, metric := range i.indexMetrics {
		ch <- metric.Desc
	}
	ch <- i.up.Desc()
	ch <- i.totalScrapes.Desc()
	ch <- i.jsonParseFailures.Desc()
}

func (c *Indices) fetchAndDecodeIndexStats() (indexStatsResponse, error) {
	var isr indexStatsResponse

	u := *c.url
	u.Path = "/_all/_stats"

	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return isr, fmt.Errorf("failed to create HTTP request: %s", err)
	}
	if SignElasticsearchRequest {
		awsauth.Sign(req)
	}

	res, err := c.client.Do(req)
	if err != nil {
		return isr, fmt.Errorf("failed to get index stats from %s://%s:%s%s: %s",
			u.Scheme, u.Hostname(), u.Port(), u.Path, err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return isr, fmt.Errorf("HTTP Request failed with code %d", res.StatusCode)
	}

	if err := json.NewDecoder(res.Body).Decode(&isr); err != nil {
		c.jsonParseFailures.Inc()
		return isr, err
	}

	return isr, nil
}

func (i *Indices) Collect(ch chan<- prometheus.Metric) {
	i.totalScrapes.Inc()
	defer func() {
		ch <- i.up
		ch <- i.totalScrapes
		ch <- i.jsonParseFailures
	}()

	// indices
	indexStatsResponse, err := i.fetchAndDecodeIndexStats()
	if err != nil {
		i.up.Set(0)
		level.Warn(i.logger).Log(
			"msg", "failed to fetch and decode index stats",
			"err", err,
		)
		return
	}
	i.up.Set(1)

	// Index stats
	for indexName, indexStats := range indexStatsResponse.Indices {
		for _, metric := range i.indexMetrics {
			ch <- prometheus.MustNewConstMetric(
				metric.Desc,
				metric.Type,
				metric.Value(indexStats),
				metric.Labels(indexName)...,
			)
		}
	}
}
