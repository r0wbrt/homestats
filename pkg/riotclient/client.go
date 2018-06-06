//Copyright Robert C. Taylor 2018
//Distributed under the terms of the LICENSE file

package riotclient

import (
	"context"
	"encoding/json"
	"fmt"
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

type RiotServer struct {

	//The path to RIOT server
	URL string

	//Name is the human readable name of the server
	Name string

	//Description is the human readable description of the server
	Description string

	//GUID is the global unique identifier associated with this node
	GUID string

	//Cached stream info
	streams map[string]serverStream
}

type riotServerResp struct {
	Name        string              `json:"name"`
	Description string              `json:"description"`
	GUID        string              `json:"GUID"`
	Links       *jsonhal.Collection `json:"_links"`
}

//Initialize connects to the riot server getting the list of
//resources present on the server.
func Initialize(ctx context.Context, serverURL string) (*RiotServer, error) {

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

	rs := RiotServer{Name: jResp.Name, Description: jResp.Description, GUID: jResp.GUID, URL: serverURL}

	rs.streams = make(map[string]serverStream)

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

			rs.streams[GUID] = serverStream{uRL: href}
		}
	}

	return &rs, nil
}

func (rs *RiotServer) GetResourceList(ctx context.Context) ([]string, error) {

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

func (rs *RiotServer) GetResource(ctx context.Context, GUID string) (stream.Stream, error) {

	ret := stream.Stream{}

	serverS, ok := rs.streams[GUID] //Check Cache
	if ok {
		if serverS.stream != nil {
			return *serverS.stream, nil
		}
	} else {
		return ret, fmt.Errorf("riotclient : ")
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

	return ret, nil
}

func (rs *RiotServer) ReadDataset() error { return nil } //TODO - Fix Type

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
