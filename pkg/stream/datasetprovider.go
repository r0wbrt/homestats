//Copyright Robert C. Taylor 2018
//Distributed under the terms of the LICENSE file

package stream

import (
	"context"
	"encoding/csv"
	"fmt"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/r0wbrt/riot/pkg/jsonhal"
)

//DataSetValue is a single value in a measurment
type DataSetValue struct {
	Name  string
	Value string
}

//DataSetMeasurment is a single measurment in a data set.
type DataSetMeasurment struct {
	Time   time.Time
	Values []DataSetValue
}

//DataSetProvider is used by StreamEndPoint to serve data requests to the client.
type DataSetProvider interface {

	//ReadRange provides a range of data. The data is returned to the client via the writer.
	ReadRange(ctx context.Context, start, end time.Time, writer DataSetWriter) error
}

//DataSetWriter provides the API to write the stream measurments to the client.
type DataSetWriter interface {
	//Write takes a set of measurments to send to the client. Returns an error
	//if something went wrong. If this does return an error, the consuming code should
	//quit and return control back to the stream reader end point.
	Write(measurments []DataSetMeasurment) error
}

//DataSetEndPoint provides a HATEOS like API over http for
//consuming stream data.
type DataSetEndPoint struct {
	//Stream has the meta data describing the stream at this end point.
	Stream *Stream
	//DataSource supplies the data.
	DataSource DataSetProvider
	//Handler is a standard HTTP handler that can be used to override the behavior of the stream.
	Handler http.HandlerFunc
}

type streamJSONReplyGetDefault struct {
	Name            string                   `json:"name,omitempty"`
	Description     string                   `json:"description,omitempty"`
	GUID            string                   `json:"guid"`
	Links           *jsonhal.Collection      `json:"_links,omitempty"`
	RetentionPolicy int64                    `json:"retentionPolicy,omitempty"`
	Schema          []*streamJSONReplySchema `json:"schema"`
}

type streamJSONReplySchema struct {
	Name           string      `json:"name"`
	StorageUnit    StorageType `json:"storageUnit"`
	MeasurmentUnit string      `json:"measurmentUnit,omitempty"`
}

//ServeHTTP services a HTTP request to the end point.
func (endpoint *DataSetEndPoint) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	//Run the handler if it is not nil
	if endpoint.Handler != nil {
		endpoint.Handler(w, r)
		return
	}

	//Only GET is supported.
	if r.Method != http.MethodGet {
		endpoint.writeReply(w, r, http.StatusMethodNotAllowed, &jsonErrorResponse{
			Detail: "The requested http method is not allowed on this resource",
			Status: http.StatusMethodNotAllowed,
			Title:  http.StatusText(http.StatusMethodNotAllowed),
		})
		msg := fmt.Sprintf("Data Set End Point : Request with invalid method recieved on %s from %s", r.URL.Path, r.RemoteAddr)
		LogErrorMessage(r, msg)
		return
	}

	//Serve the reply using a http mux
	mux := http.NewServeMux()
	mux.HandleFunc(fixSlash(path.Join(GetPathPrefix(r), endpoint.Stream.GUID)), endpoint.rootReply)
	mux.HandleFunc(path.Join(GetPathPrefix(r), endpoint.Stream.GUID, "dataset"), endpoint.datasetReply)

	mux.ServeHTTP(w, r)
}

func (endpoint *DataSetEndPoint) datasetReply(w http.ResponseWriter, r *http.Request) {

	acceptHeader := r.Header.Get("Accept")

	//Check if the client supports the format we will send the reply in
	if !strings.Contains(acceptHeader, "text/csv") && !strings.Contains(acceptHeader, "*/*") && !strings.Contains(acceptHeader, "text/*") {
		endpoint.writeReply(w, r, http.StatusExpectationFailed, &jsonErrorResponse{
			Detail: "The expected type could not be fullfilled.",
			Status: http.StatusExpectationFailed,
			Title:  http.StatusText(http.StatusExpectationFailed),
		})
		return
	}

	//Parse the input
	startStr := r.FormValue("start")
	endStr := r.FormValue("end")
	var err error
	var start, end time.Time
	if startStr != "" {
		start, err = time.Parse(time.RFC3339Nano, startStr)
		if err != nil {
			endpoint.invalidTimeFormatError(w, r)
			return
		}
	}

	if endStr != "" {
		end, err = time.Parse(time.RFC3339Nano, endStr)
		if err != nil {
			endpoint.invalidTimeFormatError(w, r)
			return
		}
	}

	//Write the header, the client is expected to support chunked responses.
	w.Header().Set("content-type", "text/csv")
	w.WriteHeader(http.StatusOK)

	//Set up DataWriter
	csvW := streamDataWriterCSV{}
	csvW.schema = endpoint.Stream.Schema
	csvW.csvWriter = csv.NewWriter(w)
	csvW.context = r.Context()

	//Populate CSV header
	var names []string
	var storeUnit []string
	var measurmentUnit []string
	for i := 0; i < len(endpoint.Stream.Schema); i++ {
		scheme := endpoint.Stream.Schema[i]
		names = append(names, scheme.Name)
		storeUnit = append(storeUnit, string(scheme.StorageUnit))
		measurmentUnit = append(measurmentUnit, scheme.MeasurmentUnit)
	}

	names = append(names, "Time")
	measurmentUnit = append(measurmentUnit, "nanosecond")
	storeUnit = append(storeUnit, string(StorageTime))

	csvW.csvWriter.Write(names)
	csvW.csvWriter.Write(measurmentUnit)
	csvW.csvWriter.Write(storeUnit)
	csvW.csvWriter.Flush()

	//Pass control to the dataset reading function
	err = endpoint.DataSource.ReadRange(r.Context(), start, end, &csvW)
	if err != nil {
		LogErrorMessage(r, fmt.Sprintf("Error in serving request : %s", err.Error()))
		panic(http.ErrAbortHandler)
	}

	//Flush any residule data before we return control the http server.
	csvW.csvWriter.Flush()
}

func (endpoint *DataSetEndPoint) invalidTimeFormatError(w http.ResponseWriter, r *http.Request) {

	endpoint.writeReply(w, r, http.StatusBadRequest, &jsonErrorResponse{
		Type:   "https://www.ietf.org/rfc/rfc3339.txt",
		Detail: "The query parameters were invalid. start and end must be in RFC3339 Nano format.",
		Status: http.StatusBadRequest,
		Title:  http.StatusText(http.StatusBadRequest),
	})

}

func (endpoint *DataSetEndPoint) rootReply(w http.ResponseWriter, r *http.Request) {

	//Handle reqeusts recieved for invalid paths
	if r.URL.Path != fixSlash(path.Join(GetPathPrefix(r), endpoint.Stream.GUID)) {
		endpoint.writeReply(w, r, http.StatusNotFound, &jsonErrorResponse{
			Type:   "http://www.w3.org/Protocols/rfc2616/rfc2616-sec10.html",
			Detail: "The requested resource was not found. Check your URI and try again.",
			Status: http.StatusNotFound,
			Title:  http.StatusText(http.StatusNotFound),
		})
		return
	}

	//Set up the reply
	reply := streamJSONReplyGetDefault{
		Name:            endpoint.Stream.Name,
		Description:     endpoint.Stream.Description,
		GUID:            endpoint.Stream.GUID,
		RetentionPolicy: endpoint.Stream.RetentionPolicy.Nanoseconds(),
	}

	//Copy over schema
	for i := 0; i < len(endpoint.Stream.Schema); i++ {
		field := endpoint.Stream.Schema[i]
		reply.Schema = append(reply.Schema, &streamJSONReplySchema{Name: field.Name, MeasurmentUnit: field.MeasurmentUnit, StorageUnit: field.StorageUnit})
	}

	//Set up links
	reply.Links = jsonhal.NewCollection()
	reply.Links.Values["self"] = []*jsonhal.CollectionValue{jsonhal.CreateLink(fixSlash(path.Join(GetPathPrefix(r), endpoint.Stream.GUID)))}
	reply.Links.Values["data"] = []*jsonhal.CollectionValue{jsonhal.CreateLink(path.Join(GetPathPrefix(r), endpoint.Stream.GUID, "dataset"))}

	endpoint.writeReply(w, r, http.StatusOK, reply)
}

func (endpoint *DataSetEndPoint) writeReply(w http.ResponseWriter, r *http.Request, code int, resp interface{}) {
	err := jsonhal.WriteHalPlusJSONResp(w, code, resp)
	if err != nil {
		address := r.RemoteAddr
		path := r.URL.Path
		msg := fmt.Sprintf("RIOT Stream End Point : Error occured while writing response for %s on path %s : %s", address, path, err.Error())
		LogErrorMessage(r, msg)
		panic(http.ErrAbortHandler) //Abort the HTTP transaction
	}
}

type streamDataWriterCSV struct {
	w         http.ResponseWriter
	csvWriter *csv.Writer
	schema    []TypeSchema
	context   context.Context
}

func (csvWriter *streamDataWriterCSV) Write(measurments []DataSetMeasurment) error {

	//Check context in case the client has closed the connection
	err := csvWriter.context.Err()
	if err != nil {
		return err
	}

	//Questionable method of sending data to the client.
	//Will likely need to reevaulate in the future to see if there
	//is a more performant way of doing this.
	for i := 0; i < len(measurments); i++ {

		//For each measurment

		measurment := measurments[i]
		var fields []string

		for j := 0; j < len(csvWriter.schema); j++ {

			//Loop over the schema after getting a measurment, the CSV columns
			//will match the order of the schema.

			schema := csvWriter.schema[j]
			found := false

			for k := 0; k < len(measurment.Values); k++ {

				//Find the value that matches the given schema in our recieved dataset.

				val := measurment.Values[k]

				if val.Name == schema.Name {
					fields = append(fields, val.Value)
					found = true
					break
				}
			}

			//If not found, the field is blank.
			if !found {
				fields = append(fields, "")
			}
		}

		//Time always goes at the end.
		fields = append(fields, measurment.Time.Format(time.RFC3339Nano))

		err := csvWriter.csvWriter.Write(fields)

		if err != nil {
			return err
		}
	}

	//Flush the data to the client.
	csvWriter.csvWriter.Flush()

	return nil
}
