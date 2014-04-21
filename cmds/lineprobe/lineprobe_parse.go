package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

type Operation interface {
	Samples() int64 // number of samples to collect
	Extra() int64   // extra data related to the operation type.
	String() string // return the name of this operation.
}

type operationSlice []Operation

var calcOperations operationSlice

func (ops *operationSlice) Append(op Operation) {
	*ops = append(*ops, op)
}

func (ops *operationSlice) String() string {
	ret := ""
	for _, op := range *ops {
		ret += op.String()
	}
	return ret
}

// an operation with no calculation of values.
type NoOperation struct {
}

func newNoOperation() *NoOperation {
	return &NoOperation{}
}

func (op *NoOperation) Set(value string) error {
	calcOperations.Append(newNoOperation())
	return nil
}

func (op *NoOperation) Samples() int64 {
	return 1
}

func (op *NoOperation) Extra() int64 {
	return 0
}

func (op *NoOperation) String() string {
	return ""
}

type AvgOperation struct {
	samples int64
	extra   int64
}

func newAvgOperation(samples int64, showStdev int64) *AvgOperation {
	return &AvgOperation{samples: samples, extra: showStdev}
}

// implements Set(value string) error of flag.Value interface.
func (op *AvgOperation) Set(value string) error {
	var err error
	var showStdev bool
	var numSamples int64

	fields := strings.Split(value, ",")
	if len(fields) > 2 {
		return errors.New("Wrong number of values specified.")
	}

	numSamples, err = strconv.ParseInt(fields[0], 10, 32)
	if err != nil {
		return err
	}

	showStdev = false
	if len(fields) == 2 {
		showStdev, err = strconv.ParseBool(fields[1])
		if err != nil {
			return err
		}
	}

	if showStdev {
		// also calculate +/- stdev
		avg := newAvgOperation(numSamples, 1)
		calcOperations.Append(avg)
		avg = newAvgOperation(numSamples, -1)
		calcOperations.Append(avg)
	}
	avg := newAvgOperation(numSamples, 0)
	calcOperations.Append(avg)
	return nil
}

func (op *AvgOperation) Samples() int64 {
	return op.samples
}

func (op *AvgOperation) Extra() int64 {
	return op.extra
}

func (op *AvgOperation) String() string {
	if op.extra == 0 {
		return fmt.Sprintf("-avg(%d)", op.samples)
	} else {
		return fmt.Sprintf("-avg(%d,%d)", op.samples, op.extra)
	}
}

type PercOperation struct {
	samples    int64
	percentile int64
}

func newPercOperation(samples int64, percentile int64) *PercOperation {
	return &PercOperation{samples: samples, percentile: percentile}
}

func (op *PercOperation) String() string {
	return fmt.Sprintf("-percentile(%d,%d)", op.samples, op.percentile)
}

// implements Set(value string) error of flag.Value interface.
func (op *PercOperation) Set(value string) error {

	fields := strings.Split(value, ",")
	if len(fields) != 2 {
		return errors.New("Wrong number of values specified. Use: <samples>,<percentile>")
	}

	numSamples, err := strconv.ParseInt(fields[0], 10, 32)
	if err != nil {
		return err
	}

	percentile, err := strconv.ParseInt(fields[1], 10, 32)
	if err != nil {
		return err
	}

	perc := newPercOperation(numSamples, percentile)
	calcOperations.Append(perc)
	return nil
}

func (op *PercOperation) Samples() int64 {
	return op.samples
}

func (op *PercOperation) Extra() int64 {
	return op.percentile
}
