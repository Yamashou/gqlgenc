package scalars

import (
	"fmt"
	"io"
	"strconv"
)

type BigFloat float64

func (f BigFloat) MarshalGQL(w io.Writer) {
	_, err := w.Write([]byte(fmt.Sprintf("%f", f)))
	if err != nil {
		panic(err)
	}
}

func (f *BigFloat) UnmarshalGQL(v interface{}) error {
	stringValue, ok := v.(string)

	if ok {
		return fmt.Errorf("BigFloat must be a string")
	}

	floatValue, err := strconv.ParseFloat(stringValue, 64)
	if err == nil {
		*f = BigFloat(floatValue)
		return nil
	}

	return err
}
