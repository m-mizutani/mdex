package notion

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/mdex/pkg/domain"
	"github.com/m-mizutani/mdex/pkg/utils/dryrun"
	"github.com/m-mizutani/mdex/pkg/utils/logging"
	"github.com/m-mizutani/mdex/pkg/utils/safe"
)

// GetDatabaseProperties retrieves the property schema of a database.
// This is primarily used for testing to inspect database structure.
func (c *Client) GetDatabaseProperties(ctx context.Context, databaseID string) (map[string]DatabasePropertySchema, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, fmt.Sprintf("/databases/%s", databaseID), nil)
	if err != nil {
		return nil, goerr.Wrap(err, "getting database schema", goerr.V("databaseID", databaseID))
	}
	defer safe.Close(ctx, resp.Body)

	var db DatabaseObject
	if err := json.NewDecoder(resp.Body).Decode(&db); err != nil {
		return nil, goerr.Wrap(err, "decoding database schema")
	}

	return db.Properties, nil
}

// EnsureDatabaseProperties ensures that the specified properties exist in the database schema.
// It creates missing properties with the appropriate type (rich_text, multi_select, select).
func (c *Client) EnsureDatabaseProperties(ctx context.Context, databaseID string, properties []domain.PropertySpec) error {
	if dryrun.IsDryRun(ctx) {
		logging.From(ctx).Info("dry-run: would ensure database properties", "database_id", databaseID, "properties", properties)
		return nil
	}

	existing, err := c.GetDatabaseProperties(ctx, databaseID)
	if err != nil {
		return goerr.Wrap(err, "getting existing properties")
	}

	// Build update payload with only missing properties
	newProps := make(map[string]interface{})
	for _, prop := range properties {
		if _, ok := existing[prop.Name]; !ok {
			newProps[prop.Name] = map[string]interface{}{
				prop.Type: map[string]interface{}{},
			}
		}
	}

	if len(newProps) == 0 {
		return nil // all properties already exist
	}

	body := map[string]interface{}{
		"properties": newProps,
	}

	resp, err := c.doRequest(ctx, http.MethodPatch, fmt.Sprintf("/databases/%s", databaseID), body)
	if err != nil {
		return goerr.Wrap(err, "updating database properties", goerr.V("databaseID", databaseID))
	}
	defer safe.Close(ctx, resp.Body)

	return nil
}
