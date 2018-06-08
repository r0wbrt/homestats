//Copyright Robert C. Taylor 2018
//Distributed under the terms of the LICENSE file

package riotclient

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/r0wbrt/riot/pkg/jsonhal"

	"github.com/r0wbrt/riot/pkg/stream"
)

type serverStream struct {
	//URL of the root of this stream
	uRL string

	//URL to the dataset provider
	datasetURL string

	//Cached copy of the stream
	stream *stream.Stream //May be nil if this stream has not been requested yet
}

type RiotEndPoint struct {

	//The path to RIOT server
	URL string

	//Name is the human readable name of the server
	Name string

	//Description is the human readable description of the server
	Description string

	//GUID is the global unique identifier associated with this node
	GUID string

	//Cached stream info
	streams map[string]*serverStream
}

type riotServerResp struct {
	Name        string              `json:"name"`
	Description string              `json:"description"`
	GUID        string              `json:"GUID"`
	Links       *jsonhal.Collection `json:"_links"`
}

//Initialize connects to the riot server getting the list of
//resources present on the server.
func Initialize(ctx context.Context, serverURL string) (*RiotEndPoint, error) {

	req, err := http.NewRequest(http.MethodGet, serverURL, nil)
	if err != nil {
		return nil, err
	}

	req = req.WithContext(ctx) //Switch over the request to the supplied context

	resp, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	respData, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}

	jResp := riotServerResp{}
	err = json.Unmarshal(respData, &jResp)
	if err != nil {
		return nil, err
	}

	rs := RiotEndPoint{Name: jResp.Name, Description: jResp.Description, GUID: jResp.GUID, URL: serverURL}

	rs.streams = make(map[string]*serverStream)

	datasetLinks, ok := jResp.Links.Values["stream"]
	if ok {
		for i := 0; i < len(datasetLinks); i++ {
			v := datasetLinks[i]

			GUID, ok := getStringFromMap("guid", v.Properties)
			if !ok {
				continue
			}

			href, ok := getStringFromMap("href", v.Properties)
			if !ok {
				continue
			}

			rs.streams[GUID] = &serverStream{uRL: href}
		}
	}

	return &rs, nil
}

func (rs *RiotEndPoint) GetResourceList(ctx context.Context) ([]string, error) {

	var resources []string

	for k := range rs.streams {
		resources = append(resources, k)
	}

	return resources, nil
}

type streamResourceJSONResp struct {
	Name            string                  `json:"name"`
	GUID            string                  `json:"guid"`
	Description     string                  `json:"description"`
	Links           *jsonhal.Collection     `json:"_links"`
	RetensionPolicy int64                   `json:"retentionPolicy"`
	Schema          []streamJSONReplySchema `json:"schema"`
}

type streamJSONReplySchema struct {
	Name           string             `json:"name"`
	StorageUnit    stream.StorageType `json:"storageUnit"`
	MeasurmentUnit string             `json:"measurmentUnit,omitempty"`
}

func (rs *RiotEndPoint) GetResource(ctx context.Context, GUID string) (stream.Stream, error) {

	ret := stream.Stream{}

	serverS, ok := rs.streams[GUID]
	if ok {
		if serverS.stream != nil {
			return *serverS.stream, nil
		}
	} else {
		return ret, fmt.Errorf("riotclient : Requested stream was not found")
	}

	str, ok := rs.streams[GUID]
	if !ok {
		return ret, fmt.Errorf("riotclient : Requested stream was not found")
	}

	pathFragment := str.uRL

	path, err := buildPath(rs.URL, pathFragment)
	if err != nil {
		return ret, err
	}

	req, err := http.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return ret, err
	}

	req = req.WithContext(ctx)

	resp, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		return ret, err
	}

	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return ret, err
	}

	var jsonReply streamResourceJSONResp

	err = json.Unmarshal(data, &jsonReply)
	if err != nil {
		return ret, err
	}

	ret.Name = jsonReply.Name
	ret.Description = jsonReply.Description
	ret.GUID = jsonReply.GUID
	ret.RetentionPolicy = time.Nanosecond * time.Duration(jsonReply.RetensionPolicy)

	for i := 0; i < len(jsonReply.Schema); i++ {
		schemaType := stream.TypeSchema{}
		schemaType.Name = jsonReply.Schema[i].Name
		schemaType.MeasurmentUnit = jsonReply.Schema[i].MeasurmentUnit
		schemaType.StorageUnit = jsonReply.Schema[i].StorageUnit
		ret.Schema = append(ret.Schema, schemaType)
	}

	serverS.stream = &ret //Cache stream for later use

	//Get dateset URL
	links, ok := jsonReply.Links.Values["data"]
	if ok && len(links) > 0 {
		relpath, ok := getStringFromMap("href", links[0].Properties)
		if ok {
			serverS.datasetURL, err = buildPath(path, relpath)
			if err != nil {
				return ret, err
			}
		}
	}

	return ret, nil
}

type DatasetReader interface {
	Read(context.Context, []stream.DataSetMeasurment) error
}

func (rs *RiotEndPoint) ReadDataset(ctx context.Context, GUID string, reader DatasetReader, start time.Time, end time.Time) error {

	_, err := rs.GetResource(ctx, GUID)
	if err != nil {
		return err
	}

	s := rs.streams[GUID]

	if s.datasetURL == "" {
		return fmt.Errorf("riotclient : Resource has not specified dataset URL")
	}

	URL, err := url.Parse(s.datasetURL)
	if err != nil {
		return err
	}

	URL.Query().Add("start", start.Format(time.RFC3339Nano))
	URL.Query().Add("end", end.Format(time.RFC3339Nano))

	req, err := http.NewRequest(http.MethodGet, URL.String(), nil)
	if err != nil {
		return err
	}

	req = req.WithContext(ctx)
	req.Header.Add("Accept", "text/csv")

	resp, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	csvReader := csv.NewReader(resp.Body)

	//Return err if an err occurs. We should not encounter an io.EOF
	//this early in the stream. If we do, then this stream is malformed.
	header, err := csvReader.Read()
	if err != nil {
		return err
	}
	_, err = csvReader.Read()
	if err != nil {
		return err
	}
	_, err = csvReader.Read()
	if err != nil {
		return err
	}

	for {
		record, err := csvReader.Read()
		if err != nil {
			if err != io.EOF {
				return err
			}
			break
		}

		values := []stream.DataSetValue{}
		for i := 0; i < len(record)-1; i++ {
			values = append(values, stream.DataSetValue{Name: header[i], Value: record[i]})
		}

		timeResult, err := time.Parse(time.RFC3339Nano, record[len(record)-1])
		if err != nil {
			return err
		}
		measurment := stream.DataSetMeasurment{Time: timeResult, Values: values}
		err = reader.Read(ctx, []stream.DataSetMeasurment{measurment})
		if err != nil {
			return err
		}
	}

	return nil
}

func buildPath(base string, ref string) (string, error) {

	baseURL, err := url.Parse(base)
	if err != nil {
		return "", err
	}

	pathURL, err := url.Parse(ref)
	if err != nil {
		return "", err
	}

	reqURL := baseURL.ResolveReference(pathURL)

	return reqURL.String(), nil
}

func getStringFromMap(key string, keyvalmap map[string]interface{}) (string, bool) {
	valueI, ok := keyvalmap[key]
	if !ok {
		return "", false
	}

	value, ok := valueI.(string)

	return value, ok
}
