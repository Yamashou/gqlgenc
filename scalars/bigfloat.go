package scalars

import (
	"encoding/json"
	"fmt"
	"strconv"
)

type BigFloat float64

func (f *BigFloat) UnmarshalJSON(b []byte) error {
	var v interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}

	switch v := v.(type) {
	case string:
		floatValue, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return nil
		}
		*f = BigFloat(floatValue)

	case int:
		*f = BigFloat(v)
	case int64:
		*f = BigFloat(v)
	case float64:
		*f = BigFloat(v)
	case json.Number:
		floatValue, err := strconv.ParseFloat(string(v), 64)
		if err != nil {
			return nil
		}
		*f = BigFloat(floatValue)
	default:
		return fmt.Errorf("%T is not a big float", v)
	}

	return nil
}

func (f BigFloat) MarshalJSON() ([]byte, error) {
	return json.Marshal(float64(f))
}
