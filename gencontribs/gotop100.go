package main

import (
	"bytes"
	_ "embed"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strconv"
)

//go:embed data/go_top_100.csv
var csvGoTop100 []byte

type placeDependent struct {
	place          int
	dependentCount int
}

type goTop100 struct {
	places map[string]placeDependent
}

func NewGoTop100() (*goTop100, error) {
	places := map[string]placeDependent{}
	r := csv.NewReader(bytes.NewReader(csvGoTop100))
	record, err := r.Read() // read header
	if err != nil {
		return nil, err
	}
	for i := 1; record != nil; i++ {
		record, err = r.Read()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}
		if len(record) != 2 {
			return nil, fmt.Errorf("unexpected number of fields in record: %q", record)
		}
		moduleName := record[0]
		depCount, err := strconv.Atoi(record[1])
		if err != nil {
			return nil, fmt.Errorf("parse dependent count %q: %w", record[1], err)
		}
		places[moduleName] = placeDependent{
			place:          i,
			dependentCount: depCount,
		}
	}
	return &goTop100{
		places: places,
	}, nil
}

func (gt goTop100) Place(moduleName string) int {
	return gt.places[moduleName].place
}
