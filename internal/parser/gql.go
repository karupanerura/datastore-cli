package parser

import (
	"fmt"

	"github.com/karupanerura/datastore-cli/internal/datastore"
	"github.com/karupanerura/gqlparser"
)

type QueryParser struct {
	Namespace string
}

func (p *QueryParser) ParseGQL(query string) (*datastore.Query, *datastore.AggregationQuery, error) {
	q, aq, err := gqlparser.ParseQueryOrAggregationQuery(gqlparser.NewLexer(query))
	if err != nil {
		return nil, nil, fmt.Errorf("gqlparser.ParseQueryOrAggregationQuery: %w", err)
	}

	if aq != nil {
		q = &aq.Query
	}

	dq := datastore.NewQuery(string(q.Kind))
	if q.Distinct {
		dq = dq.Distinct()
	}
	if len(q.DistinctOn) != 0 {
		props := make([]string, len(q.DistinctOn))
		for i, p := range q.DistinctOn {
			props[i] = string(p)
		}
		dq = dq.DistinctOn(props...)
	}
	if q.Properties != nil {
		props := make([]string, len(q.Properties))
		for i, p := range q.Properties {
			props[i] = string(p)
		}
		dq = dq.Project(props...)
	}
	if q.Where != nil {
		filterParser := &FilterParser{Namespace: p.Namespace}
		ancestor, filter, err := filterParser.convertCondition(q.Where)
		if err != nil {
			return nil, nil, fmt.Errorf("filterParser.ParseFilter: %w", err)
		}
		if ancestor != nil {
			dq = dq.Ancestor(ancestor.ToDatastore())
		}
		dq = dq.FilterEntity(filter)
	}
	for _, order := range q.OrderBy {
		if order.Descending {
			dq = dq.Order("-" + string(order.Property))
		} else {
			dq = dq.Order(string(order.Property))
		}
	}
	if q.Limit != nil {
		dq = dq.Limit(int(q.Limit.Position))
	}
	if q.Offset != nil {
		dq = dq.Offset(int(q.Offset.Position))
	}
	if aq != nil {
		daq := dq.NewAggregationQuery()
		for _, agg := range aq.Aggregations {
			switch agg := agg.(type) {
			case *gqlparser.CountAggregation:
				daq = daq.WithCount(agg.Alias)
			case *gqlparser.SumAggregation:
				daq = daq.WithSum(string(agg.Property), agg.Alias)
			case *gqlparser.AvgAggregation:
				daq = daq.WithAvg(string(agg.Property), agg.Alias)
			default:
				return nil, nil, fmt.Errorf("unexpected aggregation: %T", agg)
			}
		}
		return nil, daq, nil
	}
	return dq, nil, nil
}
