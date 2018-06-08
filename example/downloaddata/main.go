package main

import (
	"context"
	"flag"
	"fmt"
	"time"

	"github.com/r0wbrt/riot/pkg/riotclient"
	"github.com/r0wbrt/riot/pkg/stream"
)

type reader struct {
}

func (r *reader) Read(ctx context.Context, data []stream.DataSetMeasurment) error {
	fmt.Print(data)
	return nil
}

func main() {

	location := flag.String("location", "", "Location is the affress of the rIOT server to query.")
	guid := flag.String("guid", "", "GUID is the id of the resource to load.")
	start := flag.String("start", "", "start is the time to start loading data from.")
	end := flag.String("end", "", "end is the time to load up until.")
	flag.Parse()

	startDate, err := time.ParseInLocation("01/02/2006 15:04:05 MST", *start, time.Local)
	if err != nil {
		panic(err)
	}

	endDate, err := time.ParseInLocation("01/02/2006 15:04:05 MST", *end, time.Local)
	if err != nil {
		panic(err)
	}

	ep, err := riotclient.Initialize(context.Background(), *location)
	if err != nil {
		panic(err)
	}

	err = ep.ReadDataset(context.Background(), *guid, &reader{}, startDate, endDate)
	if err != nil {
		panic(err)
	}

	return

}
