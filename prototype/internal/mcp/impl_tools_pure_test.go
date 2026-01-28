package mcp

import "testing"

func TestValidateNoExtraArgsForSchema(t *testing.T) {
	tests := []struct {
		name    string
		schema  map[string]interface{}
		args    map[string]interface{}
		wantErr bool
	}{
		{
			name: "empty args and empty schema",
			schema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
			args:    map[string]interface{}{},
			wantErr: false,
		},
		{
			name: "args match schema properties",
			schema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]string{"type": "string"},
					"age":  map[string]string{"type": "integer"},
				},
			},
			args: map[string]interface{}{
				"name": "John",
				"age":  30,
			},
			wantErr: false,
		},
		{
			name: "extra arg not in schema",
			schema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]string{"type": "string"},
				},
			},
			args: map[string]interface{}{
				"name":  "John",
				"extra": "value",
			},
			wantErr: true,
		},
		{
			name: "no properties in schema accepts anything",
			schema: map[string]interface{}{
				"type": "object",
			},
			args: map[string]interface{}{
				"anything": "goes",
			},
			wantErr: false,
		},
		{
			name: "nil properties accepts anything",
			schema: map[string]interface{}{
				"type":       "object",
				"properties": nil,
			},
			args: map[string]interface{}{
				"anything": "goes",
			},
			wantErr: false,
		},
		{
			name: "empty args with valid schema",
			schema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]string{"type": "string"},
				},
			},
			args:    map[string]interface{}{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a minimal ToolRegistry for testing
			registry := &ToolRegistry{
				tools: make(map[string]*ToolWrapper),
			}
			err := registry.validateNoExtraArgsForSchema(tt.schema, tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateNoExtraArgsForSchema() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
