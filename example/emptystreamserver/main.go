//Copyright Robert C. Taylor 2018
//Distributed under the terms of the LICENSE file

package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/r0wbrt/riot/pkg/riotserver"
)

type StreamDataSource struct {
}

func (s *StreamDataSource) ReadRange(ctx context.Context, start, end time.Time, writer riotserver.DataSetWriter) error {

	var v1 = riotserver.DataSetValue{}
	v1.Value = "1.2"
	v1.Name = "Flouride"

	var v2 = riotserver.DataSetValue{}
	v2.Value = "1.1"
	v2.Name = "Water"

	var m = riotserver.DataSetMeasurment{Values: []riotserver.DataSetValue{v1, v2}}

	writer.Write([]riotserver.DataSetMeasurment{m})

	return nil
}

func main() {
	halServer := &riotserver.Server{
		Name:        "Empty Server",
		Description: "This server has empty streams for clients to consume.",
		GUID:        "FFFFFFFF00000001",
		PathPrefix:  "/",
	}

	halServer.Streams = append(halServer.Streams, &riotserver.DataSetEndPoint{
		Stream: &riotserver.Stream{
			Name:            "Empty Test Stream 1",
			Description:     "This stream has no data",
			GUID:            "FFFFFFFF01000001",
			RetentionPolicy: time.Second * 120,
			Schema: []riotserver.TypeSchema{
				riotserver.TypeSchema{Name: "Water", StorageUnit: riotserver.StorageNumber, MeasurmentUnit: "gpm"},
				riotserver.TypeSchema{Name: "Flouride", StorageUnit: riotserver.StorageNumber, MeasurmentUnit: "ppm"},
			},
		},
		DataSource: &StreamDataSource{},
	})

	halServer.Streams = append(halServer.Streams, &riotserver.DataSetEndPoint{Stream: &riotserver.Stream{
		Name: "Empty Test Stream 2",
		GUID: "FFFFFFFF01000002",
	}})

	httpServer := http.Server{Handler: halServer, Addr: ":7468"}

	err := httpServer.ListenAndServe()
	log.Fatal(err)
}
