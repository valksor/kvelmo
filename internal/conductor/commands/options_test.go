package commands

import "testing"

func TestDecodeOptions(t *testing.T) {
	type opts struct {
		Name  string `json:"name"`
		Count int    `json:"count"`
		Flag  bool   `json:"flag"`
	}

	got, err := DecodeOptions[opts](Invocation{
		Options: map[string]any{
			"name":  "alpha",
			"count": 3,
			"flag":  true,
		},
	})
	if err != nil {
		t.Fatalf("DecodeOptions returned error: %v", err)
	}
	if got.Name != "alpha" || got.Count != 3 || !got.Flag {
		t.Fatalf("unexpected decoded options: %#v", got)
	}
}

func TestOptionGetters(t *testing.T) {
	opts := map[string]any{
		"s":  "value",
		"b":  true,
		"i1": 7,
		"i2": float64(8),
		"i3": "9",
	}

	if got := GetString(opts, "s"); got != "value" {
		t.Fatalf("GetString = %q", got)
	}
	if !GetBool(opts, "b") {
		t.Fatalf("GetBool should return true")
	}
	if got := GetInt(opts, "i1"); got != 7 {
		t.Fatalf("GetInt(i1) = %d", got)
	}
	if got := GetInt(opts, "i2"); got != 8 {
		t.Fatalf("GetInt(i2) = %d", got)
	}
	if got := GetInt(opts, "i3"); got != 9 {
		t.Fatalf("GetInt(i3) = %d", got)
	}
}
