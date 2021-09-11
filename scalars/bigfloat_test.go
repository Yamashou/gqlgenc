package scalars

import (
	"encoding/json"
	"testing"
)

func TestUnmarshalBigFloat(t *testing.T) {
	t.Parallel()
	var bf BigFloat
	err := json.Unmarshal([]byte(`"2.0"`), &bf)
	if err != nil {
		t.Fatalf(err.Error())
	}

	if float64(bf) != 2.0 {
		t.Fatalf("bf is %v", bf)
	}

	err = json.Unmarshal([]byte(`2000`), &bf)
	if err != nil {
		t.Fatalf(err.Error())
	}
	if float64(2000) != float64(bf) {
		t.Fatalf("bf is %v", bf)
	}
}
