package linear

import (
	"testing"
	"time"
)

func TestExtractLabelNames(t *testing.T) {
	tests := []struct {
		name   string
		labels *LabelConnection
		want   []string
	}{
		{
			name:   "nil labels",
			labels: nil,
			want:   []string{},
		},
		{
			name:   "empty labels",
			labels: &LabelConnection{},
			want:   []string{},
		},
		{
			name: "multiple labels",
			labels: &LabelConnection{
				Nodes: []*Label{
					{Name: "backend"},
					{Name: "urgent"},
				},
			},
			want: []string{"backend", "urgent"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractLabelNames(tt.labels)
			if len(got) != len(tt.want) {
				t.Fatalf("len(extractLabelNames()) = %d, want %d", len(got), len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("extractLabelNames()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestMapAttachments(t *testing.T) {
	createdAt := time.Now().UTC().Truncate(time.Second)
	attachments := []*Attachment{
		{
			ID:        "att-1",
			Title:     "screenshot.png",
			URL:       "https://example.com/1",
			CreatedAt: createdAt,
		},
	}

	got := mapAttachments(attachments)
	if len(got) != 1 {
		t.Fatalf("len(mapAttachments()) = %d, want 1", len(got))
	}
	if got[0].ID != "https://example.com/1" {
		t.Fatalf("Attachment ID = %q, want URL", got[0].ID)
	}
	if got[0].Name != "screenshot.png" {
		t.Fatalf("Attachment Name = %q, want screenshot.png", got[0].Name)
	}
	if !got[0].CreatedAt.Equal(createdAt) {
		t.Fatalf("Attachment CreatedAt = %v, want %v", got[0].CreatedAt, createdAt)
	}
}

func TestMapAssignees(t *testing.T) {
	t.Run("nil assignee", func(t *testing.T) {
		got := mapAssignees(nil)
		if len(got) != 0 {
			t.Fatalf("len(mapAssignees(nil)) = %d, want 0", len(got))
		}
	})

	t.Run("single assignee", func(t *testing.T) {
		got := mapAssignees(&User{
			ID:    "u1",
			Name:  "Ada",
			Email: "ada@example.com",
		})
		if len(got) != 1 {
			t.Fatalf("len(mapAssignees()) = %d, want 1", len(got))
		}
		if got[0].Name != "Ada" {
			t.Fatalf("Assignee name = %q, want Ada", got[0].Name)
		}
	})
}

func TestMapComments(t *testing.T) {
	t.Run("nil comments", func(t *testing.T) {
		if got := mapComments(nil); got != nil {
			t.Fatalf("mapComments(nil) = %#v, want nil", got)
		}
	})

	t.Run("author and anonymous comments", func(t *testing.T) {
		now := time.Now().UTC().Truncate(time.Second)
		got := mapComments([]*Comment{
			{
				ID:        "c1",
				Body:      "first",
				CreatedAt: now,
				UpdatedAt: now,
				User: &User{
					ID:   "u1",
					Name: "Author",
				},
			},
			{
				ID:        "c2",
				Body:      "second",
				CreatedAt: now,
				UpdatedAt: now,
			},
		})

		if len(got) != 2 {
			t.Fatalf("len(mapComments()) = %d, want 2", len(got))
		}
		if got[0].Author.Name != "Author" {
			t.Fatalf("first author name = %q, want Author", got[0].Author.Name)
		}
		if got[1].Author.Name != "" {
			t.Fatalf("anonymous comment author name = %q, want empty", got[1].Author.Name)
		}
	})
}

func TestBuildMetadata(t *testing.T) {
	issueWithTeam := &Issue{
		Identifier: "ENG-1",
		URL:        "https://linear.app/team/issue/ENG-1",
		State: &State{
			ID:   "state-1",
			Name: "In Progress",
			Type: "started",
		},
		Team: &Team{
			Key:  "ENG",
			Name: "Engineering",
		},
	}

	meta := buildMetadata(issueWithTeam)
	if meta["identifier"] != "ENG-1" {
		t.Fatalf("identifier = %v, want ENG-1", meta["identifier"])
	}
	if meta["team_key"] != "ENG" {
		t.Fatalf("team_key = %v, want ENG", meta["team_key"])
	}

	issueWithoutTeam := &Issue{
		Identifier: "ENG-2",
		URL:        "https://linear.app/team/issue/ENG-2",
		State: &State{
			ID:   "state-2",
			Name: "Todo",
			Type: "unstarted",
		},
	}
	meta = buildMetadata(issueWithoutTeam)
	if _, exists := meta["team_key"]; exists {
		t.Fatalf("team_key should be absent when team is nil")
	}
}

func TestPriorityLabel(t *testing.T) {
	tests := []struct {
		want string
		in   int
	}{
		{in: 1, want: "Urgent"},
		{in: 2, want: "High"},
		{in: 3, want: "Medium"},
		{in: 4, want: "Low"},
		{in: 0, want: "No priority"},
	}

	for _, tt := range tests {
		if got := priorityLabel(tt.in); got != tt.want {
			t.Fatalf("priorityLabel(%d) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestMatchesLabels(t *testing.T) {
	issue := &Issue{
		Labels: &LabelConnection{
			Nodes: []*Label{
				{Name: "bug"},
				{Name: "backend"},
			},
		},
	}

	if !matchesLabels(issue, nil) {
		t.Fatalf("expected nil filter to match")
	}
	if !matchesLabels(issue, []string{"bug"}) {
		t.Fatalf("expected single matching label filter to match")
	}
	if matchesLabels(issue, []string{"bug", "frontend"}) {
		t.Fatalf("expected missing label to fail match")
	}
}
