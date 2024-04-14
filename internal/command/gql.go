package command

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"strings"

	"github.com/karupanerura/datastore-cli/internal/datastore"
	"github.com/karupanerura/datastore-cli/internal/parser"
	"google.golang.org/api/iterator"
)

type GQLCommand struct {
	Query string `arg:"" name:"query" help:"GQL Query"`
}

func (r *GQLCommand) Run(ctx context.Context, opts Options) error {
	client, err := datastore.NewClient(ctx, opts.Datastore())
	if err != nil {
		return err
	}

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
		b, err := json.Marshal(props)
		if err != nil {
			return err
		}

		_, err = io.Copy(os.Stdout, io.MultiReader(bytes.NewReader(b), strings.NewReader("\n")))
		if err != nil {
			return err
		}
		return nil
	}

	iter := client.Run(ctx, q)
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
