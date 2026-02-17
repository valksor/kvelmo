package notion

import (
	"context"
	"strings"

	"github.com/valksor/go-toolkit/workunit"
)

// AddComment adds a comment to a Notion page.
func (p *Provider) AddComment(ctx context.Context, workUnitID, body string) (*workunit.Comment, error) {
	input := &AddCommentInput{
		Parent: CommentParent{
			BlockID: workUnitID,
		},
		RichText: []RichText{
			{
				Type: "text",
				Text: &TextContent{
					Content: body,
				},
				PlainText: body,
			},
		},
	}

	comment, err := p.client.AddComment(ctx, input)
	if err != nil {
		return nil, err
	}

	return &workunit.Comment{
		ID:        comment.ID,
		Body:      body,
		CreatedAt: comment.CreatedTime,
		UpdatedAt: comment.LastEditedTime,
	}, nil
}

// FetchComments retrieves comments for a Notion page.
func (p *Provider) FetchComments(ctx context.Context, id string) ([]workunit.Comment, error) {
	comments, err := p.client.GetComments(ctx, id)
	if err != nil {
		return nil, err
	}

	result := make([]workunit.Comment, 0, len(comments))
	for _, c := range comments {
		// Extract comment text from rich text
		body := ""
		var bodySb45 strings.Builder
		for _, rt := range c.RichText {
			bodySb45.WriteString(rt.PlainText)
		}
		body += bodySb45.String()

		// Extract author info from created_by (which is a RichText in Notion's API)
		var author workunit.Person
		if c.CreatedBy.Type == "person" && len(c.RichText) > 0 {
			// Notion's comment API returns limited user info
			author = workunit.Person{
				Name: "Notion User",
			}
		}

		result = append(result, workunit.Comment{
			ID:        c.ID,
			Body:      body,
			CreatedAt: c.CreatedTime,
			UpdatedAt: c.LastEditedTime,
			Author:    author,
		})
	}

	return result, nil
}
