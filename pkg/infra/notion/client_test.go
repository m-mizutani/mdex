package notion_test

import (
	"context"
	"os"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/mdex/pkg/domain"
	"github.com/m-mizutani/mdex/pkg/domain/converter"
	"github.com/m-mizutani/mdex/pkg/infra/notion"
)

func setupClient(t *testing.T) (*notion.Client, string) {
	t.Helper()

	token := os.Getenv("TEST_NOTION_TOKEN")
	databaseID := os.Getenv("TEST_NOTION_DATABASE_ID")
	if token == "" || databaseID == "" {
		t.Skip("TEST_NOTION_TOKEN and TEST_NOTION_DATABASE_ID are required")
	}

	return notion.New(token), databaseID
}

// cleanupPage registers a cleanup function that archives the given page when the test ends.
func cleanupPage(t *testing.T, client *notion.Client, pageID string) {
	t.Helper()
	t.Cleanup(func() {
		if err := client.ArchivePage(context.Background(), pageID); err != nil {
			t.Logf("failed to cleanup page %s: %v", pageID, err)
		}
	})
}

// createTestPage creates a minimal page in the database with only a title.
// Returns the page ID. Registers cleanup to archive the page on test end.
func createTestPage(t *testing.T, client *notion.Client, databaseID string, title string) string {
	t.Helper()
	ctx := context.Background()

	titleProp, err := client.GetTitleProperty(ctx, databaseID)
	gt.NoError(t, err).Required()

	properties := map[string]interface{}{
		titleProp: map[string]interface{}{
			"title": []map[string]interface{}{
				{
					"type": "text",
					"text": map[string]interface{}{
						"content": title,
					},
				},
			},
		},
	}

	pageID, err := client.CreatePage(ctx, databaseID, properties, nil)
	gt.NoError(t, err).Required()
	gt.S(t, pageID).IsNotEmpty()
	cleanupPage(t, client, pageID)

	return pageID
}

func TestGetTitleProperty(t *testing.T) {
	client, databaseID := setupClient(t)
	ctx := context.Background()

	title, err := client.GetTitleProperty(ctx, databaseID)
	gt.NoError(t, err)
	gt.S(t, title).IsNotEmpty()
}

func TestGetDatabaseProperties(t *testing.T) {
	client, databaseID := setupClient(t)
	ctx := context.Background()

	props, err := client.GetDatabaseProperties(ctx, databaseID)
	gt.NoError(t, err)
	gt.N(t, len(props)).Greater(0)

	// Verify at least a title property exists
	var hasTitleProp bool
	for _, prop := range props {
		if prop.Type == "title" {
			hasTitleProp = true
			break
		}
	}
	gt.B(t, hasTitleProp).True()
}

func TestQueryDatabaseWithCustomProperties(t *testing.T) {
	client, databaseID := setupClient(t)
	ctx := context.Background()

	// Ensure the database has mdex_path and mdex_hash rich_text properties
	err := client.EnsureDatabaseProperties(ctx, databaseID, []domain.PropertySpec{
		{Name: "mdex_path", Type: "rich_text"},
		{Name: "mdex_hash", Type: "rich_text"},
	})
	gt.NoError(t, err).Required()

	// Create a page with custom properties set
	titleProp, err := client.GetTitleProperty(ctx, databaseID)
	gt.NoError(t, err).Required()

	testPath := "test/query_test.md"
	testHash := "abc123hash"

	properties := map[string]interface{}{
		titleProp: map[string]interface{}{
			"title": []map[string]interface{}{
				{
					"type": "text",
					"text": map[string]interface{}{
						"content": "mdex query test",
					},
				},
			},
		},
		"mdex_path": map[string]interface{}{
			"rich_text": []map[string]interface{}{
				{
					"type": "text",
					"text": map[string]interface{}{
						"content": testPath,
					},
				},
			},
		},
		"mdex_hash": map[string]interface{}{
			"rich_text": []map[string]interface{}{
				{
					"type": "text",
					"text": map[string]interface{}{
						"content": testHash,
					},
				},
			},
		},
	}

	pageID, err := client.CreatePage(ctx, databaseID, properties, nil)
	gt.NoError(t, err).Required()
	cleanupPage(t, client, pageID)

	// Query the database and verify custom properties are readable
	pages, err := client.QueryDatabase(ctx, databaseID, "mdex_path", "mdex_hash", "", "")
	gt.NoError(t, err).Required()

	// Find our page in results
	var found bool
	for _, page := range pages {
		if page.ID == pageID {
			found = true
			gt.S(t, page.MdexPath).Equal(testPath)
			gt.S(t, page.MdexHash).Equal(testHash)
			break
		}
	}
	gt.B(t, found).True()
}

func TestCreateAndGetPage(t *testing.T) {
	client, databaseID := setupClient(t)
	ctx := context.Background()

	pageID := createTestPage(t, client, databaseID, "mdex create+get test")

	// Verify page was created via GetPage
	// Notion API returns page ID as hyphenated UUID (e.g. "b55c9c91-384d-452b-81db-d1ef79372b75")
	page, err := client.GetPage(ctx, pageID)
	gt.NoError(t, err).Required()
	gt.S(t, page.ID).Equal(pageID)
	gt.B(t, page.Archived).False()
}

func TestCreateAndArchivePage(t *testing.T) {
	client, databaseID := setupClient(t)
	ctx := context.Background()

	pageID := createTestPage(t, client, databaseID, "mdex archive test")

	// Archive the page
	err := client.ArchivePage(ctx, pageID)
	gt.NoError(t, err)

	// Verify page is archived via GetPage
	archivedPage, err := client.GetPage(ctx, pageID)
	gt.NoError(t, err).Required()
	gt.B(t, archivedPage.Archived).True()
}

func TestCreatePageWithBlocks(t *testing.T) {
	client, databaseID := setupClient(t)
	ctx := context.Background()

	titleProp, err := client.GetTitleProperty(ctx, databaseID)
	gt.NoError(t, err).Required()

	properties := map[string]interface{}{
		titleProp: map[string]interface{}{
			"title": []map[string]interface{}{
				{
					"type": "text",
					"text": map[string]interface{}{
						"content": "mdex blocks test",
					},
				},
			},
		},
	}

	blocks := []interface{}{
		map[string]interface{}{
			"type": "heading_1",
			"heading_1": map[string]interface{}{
				"rich_text": []map[string]interface{}{
					{
						"type": "text",
						"text": map[string]interface{}{
							"content": "Test Heading",
						},
					},
				},
			},
		},
		map[string]interface{}{
			"type": "paragraph",
			"paragraph": map[string]interface{}{
				"rich_text": []map[string]interface{}{
					{
						"type": "text",
						"text": map[string]interface{}{
							"content": "This page was created by an integration test.",
						},
					},
				},
			},
		},
	}

	pageID, err := client.CreatePage(ctx, databaseID, properties, blocks)
	gt.NoError(t, err).Required()
	gt.S(t, pageID).IsNotEmpty()
	cleanupPage(t, client, pageID)

	page, err := client.GetPage(ctx, pageID)
	gt.NoError(t, err).Required()
	gt.S(t, page.ID).Equal(pageID)
	gt.B(t, page.Archived).False()
}

func TestCreatePageWithManyBlocks(t *testing.T) {
	client, databaseID := setupClient(t)
	ctx := context.Background()

	titleProp, err := client.GetTitleProperty(ctx, databaseID)
	gt.NoError(t, err).Required()

	properties := map[string]interface{}{
		titleProp: map[string]interface{}{
			"title": []map[string]interface{}{
				{
					"type": "text",
					"text": map[string]interface{}{
						"content": "mdex block split test",
					},
				},
			},
		},
	}

	// Create 150 blocks to test the 100-block split logic.
	// CreatePage sends first 100 via POST /v1/pages, remaining 50 via PATCH /v1/blocks/{id}/children.
	blocks := make([]interface{}, 150)
	for i := range blocks {
		blocks[i] = map[string]interface{}{
			"type": "paragraph",
			"paragraph": map[string]interface{}{
				"rich_text": []map[string]interface{}{
					{
						"type": "text",
						"text": map[string]interface{}{
							"content": "Block paragraph for split test",
						},
					},
				},
			},
		}
	}

	pageID, err := client.CreatePage(ctx, databaseID, properties, blocks)
	gt.NoError(t, err).Required()
	gt.S(t, pageID).IsNotEmpty()
	cleanupPage(t, client, pageID)

	// Verify the page exists and is not archived (confirms both create and append succeeded)
	page, err := client.GetPage(ctx, pageID)
	gt.NoError(t, err).Required()
	gt.S(t, page.ID).Equal(pageID)
	gt.B(t, page.Archived).False()
}

func TestCreatePageWithEmptyBlockquote(t *testing.T) {
	client, databaseID := setupClient(t)
	ctx := context.Background()

	titleProp, err := client.GetTitleProperty(ctx, databaseID)
	gt.NoError(t, err).Required()

	// Convert markdown containing an empty blockquote
	md := "# Test Page\n\nSome text before.\n\n>\n\nSome text after.\n"
	convertedBlocks, err := converter.Convert([]byte(md), "test.md", "")
	gt.NoError(t, err).Required()

	properties := map[string]interface{}{
		titleProp: map[string]interface{}{
			"title": []map[string]interface{}{
				{
					"type": "text",
					"text": map[string]interface{}{
						"content": "mdex empty blockquote test",
					},
				},
			},
		},
	}

	// Convert []converter.Block to []interface{}
	blocks := make([]interface{}, len(convertedBlocks))
	for i, b := range convertedBlocks {
		blocks[i] = b
	}

	// This should succeed without "rich_text should be an array, instead was null" error
	pageID, err := client.CreatePage(ctx, databaseID, properties, blocks)
	gt.NoError(t, err).Required()
	gt.S(t, pageID).IsNotEmpty()
	cleanupPage(t, client, pageID)
}
