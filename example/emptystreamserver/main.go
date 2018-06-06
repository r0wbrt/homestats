//Copyright Robert C. Taylor 2018
//Distributed under the terms of the LICENSE file

package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/handlers"

	"github.com/r0wbrt/riot/pkg/riotserver"
	"github.com/r0wbrt/riot/pkg/stream"
)

type StreamDataSource struct {
}

func (s *StreamDataSource) ReadRange(ctx context.Context, start, end time.Time, writer riotserver.DataSetWriter) error {

	var v1 = stream.DataSetValue{}
	v1.Value = "1.2"
	v1.Name = "Flouride"

	var v2 = stream.DataSetValue{}
	v2.Value = "1.1"
	v2.Name = "Water"

	var m = stream.DataSetMeasurment{Values: []stream.DataSetValue{v1, v2}}

	writer.Write([]stream.DataSetMeasurment{m})

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
		Stream: &stream.Stream{
			Name:            "Empty Test Stream 1",
			Description:     "This stream has no data",
			GUID:            "FFFFFFFF01000001",
			RetentionPolicy: time.Second * 120,
			Schema: []stream.TypeSchema{
				stream.TypeSchema{Name: "Water", StorageUnit: stream.StorageNumber, MeasurmentUnit: "gpm"},
				stream.TypeSchema{Name: "Flouride", StorageUnit: stream.StorageNumber, MeasurmentUnit: "ppm"},
			},
		},
		DataSource: &StreamDataSource{},
	})

	halServer.Streams = append(halServer.Streams, &riotserver.DataSetEndPoint{Stream: &stream.Stream{
		Name: "Empty Test Stream 2",
		GUID: "FFFFFFFF01000002",
	}})

	handler := handlers.CombinedLoggingHandler(os.Stdout, halServer)

	httpServer := http.Server{Handler: handler, Addr: ":7468"}

	err := httpServer.ListenAndServe()
	log.Fatal(err)
}
