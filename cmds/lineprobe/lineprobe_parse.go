package main

import (
	"fmt"
	"errors"
	"strconv"
	"strings"
)

type OperationType int
const (
	OpNone OperationType = iota
	OpMean
	OpStdev
	OpPercentile
)

type Operation struct {
	operation OperationType
	samples int64
	percentile int64	// optional, if needed by operation.
}

var OperationNames = map[string]OperationType {
	"avg": OpMean,
	"stdev": OpStdev,
	"perc": OpPercentile,
}

var OperationTypes = map[OperationType]string {
	OpMean: "avg",
	OpStdev: "stdev",
	OpPercentile: "perc",
}

func (op *Operation) String() string {
	if op.operation == OpPercentile {
		return fmt.Sprintf("%s,%d,%d", OperationTypes[op.operation], op.samples, op.percentile)
	} else {
		return fmt.Sprintf("%s,%d", OperationTypes[op.operation], op.samples)
	}
	return "err"
}

type operationSlice []*Operation

// implements String() of flag.Value interface.
func (ops *operationSlice) String() string {
	ret := ""
	for _, op := range *ops {
		if op.operation == OpPercentile {
			ret += fmt.Sprintf("--operation=%s,%d,%d",
													OperationTypes[op.operation],
													op.samples, op.percentile)
		} else {
			ret += fmt.Sprintf("--operation=%s,%d",
													OperationTypes[op.operation], op.samples)
		}
	}
	return ret
}

// implements Set(value string) error of flag.Value interface.
func (op *operationSlice) Set(value string) error {
	var err error
	var ok bool

	fields := strings.Split(value, ",")
	if len(fields) < 2 || len(fields) > 3 {
		return errors.New("Wrong operation specification.")
	}
	operation := Operation{}
	operation.operation, ok = OperationNames[fields[0]]
	if !ok {
		return errors.New("Wrong operation name.")
	}
	operation.samples, err = strconv.ParseInt(fields[1], 10, 32)
	if err != nil {
		return err
	}
	if len(fields) == 3 {
		operation.percentile, err = strconv.ParseInt(fields[2], 10, 32)
		if err != nil {
			return err
		}
	}
  *op = append(*op, &operation)
	return nil
}

/*

// Defines a new type for list of strings
type stringSlice []string

// implements String() of flag.Value interface.
func (s *stringSlice) String() string {
    return fmt.Sprintf("%v", *s)
}

// implements Set(value string) error of flag.Value interface.
func (s *stringSlice) Set(value string) error {
    fmt.Printf("adding %s\n", value)
    *s = append(*s, value)
    return nil
}

// Defines a new type for list of int
type intSlice []int

// implements String() of flag.Value interface.
func (i *intSlice) String() string {
    return fmt.Sprintf("%v", *i)
}

// implements Set(value string) error of flag.Value interface.
func (i *intSlice) Set(value string) error {
    fmt.Printf("adding %s\n", value)
		val, err := strconv.ParseInt(value, 10, 32)
		if err != nil {
			return err
		}
    *i = append(*i, int(val))
    return nil
}

*/
