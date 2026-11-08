package game

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
)

func decodeCommand(data []byte) (Command, error) {
	var c Command
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err := d.Decode(&c); err != nil {
		return c, err
	}
	if err := d.Decode(new(any)); err != io.EOF {
		return c, errors.New("trailing data")
	}
	return c, nil
}
