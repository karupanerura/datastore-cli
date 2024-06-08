package command

import (
	"context"
	"encoding/json"
	"os"

	"google.golang.org/api/iterator"

	"github.com/karupanerura/datastore-cli/internal/datastore"
	"github.com/karupanerura/datastore-cli/internal/parser"
)

type GQLCommand struct {
	Query   string `arg:"" name:"query" help:"GQL Query"`
	Explain bool   `name:"explain" optional:"" group:"Query" help:"Explain query execution plan"`
}

func (r *GQLCommand) Run(ctx context.Context, opts Options) error {
	client, err := datastore.NewClient(ctx, opts.Datastore())
	if err != nil {
		return err
	}
	defer client.Close()

	qp := &parser.QueryParser{Namespace: opts.Namespace}
	q, aq, err := qp.ParseGQL(r.Query)
	if err != nil {
		return err
	}
	if aq != nil {
		ar, err := client.RunAggregationQuery(ctx, aq)
		if err != nil {
			return err
		}

		props := datastore.NewPropertiesByProtoValueMap(ar)
		err = json.NewEncoder(os.Stdout).Encode(props)
		if err != nil {
			return err
		}
		return nil
	}

	options := []datastore.RunOption{}
	if r.Explain {
		options = append(options, datastore.ExplainOptions{Analyze: true})
	}

	iter := client.RunWithOptions(ctx, q, options...)
	if r.Explain {
		// read all
		for {
			if _, err := iter.Next(nil); err == iterator.Done {
				return json.NewEncoder(os.Stdout).Encode(iter.ExplainMetrics)
			} else if err != nil {
				return err
			}
		}
	}

	encoder := json.NewEncoder(os.Stdout)
	for {
		var entity datastore.Entity
		key, err := iter.Next(&entity)
		if err == iterator.Done {
			break
		} else if err != nil {
			return err
		}

		if len(entity.Properties) == 0 {
			if err := encoder.Encode(datastore.FromDatastoreKey(key)); err != nil {
				return err
			}
		} else {
			if err := encoder.Encode(entity); err != nil {
				return err
			}
		}
	}
	return nil
}
