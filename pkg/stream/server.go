//Copyright Robert C. Taylor 2018
//Distributed under the terms of the LICENSE file

package stream

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"path"
	"runtime/debug"

	"github.com/r0wbrt/riot/pkg/jsonhal"
)

//Server is an HTTP server that implements the IOT RIOT protocol for data aggregation.
type Server struct {
	PathPrefix  string
	Name        string
	Description string
	GUID        string
	Streams     []*DataSetEndPoint
	RootHandler func(w http.ResponseWriter, r *http.Request)
	ErrLogger   func(message string)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	mux := http.NewServeMux()

	//Install Root handler
	mux.HandleFunc("/", s.getRootHandler())

	//Install Stream Handlers
	for i := 0; i < len(s.Streams); i++ {
		mux.Handle(fixSlash(path.Join(s.PathPrefix, s.Streams[i].Stream.GUID, "/")), s.Streams[i])
	}

	wr := r.WithContext(context.WithValue(r.Context(), ctxKey("Server"), s))

	//Call mux
	mux.ServeHTTP(w, wr)

}

type jsonRootResponse struct {
	Name        string              `json:"name,omitempty"`
	Description string              `json:"description,omitempty"`
	GUID        string              `json:"guid"`
	Links       *jsonhal.Collection `json:"_links,omitempty"`
}

type jsonErrorResponse struct {
	Type   string `json:"type,omitempty"`
	Detail string `json:"detail,omitempty"`
	Status int    `json:"status,omitempty"`
	Title  string `json:"title,omitempty"`
}

//DefaultRoot is the main handler for this http end point.
func (s *Server) DefaultRoot(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodGet {
		s.writeReply(w, r, http.StatusMethodNotAllowed, &jsonErrorResponse{
			Detail: "The requested http method is not allowed on this resource",
			Status: http.StatusMethodNotAllowed,
			Title:  http.StatusText(http.StatusMethodNotAllowed),
		})
		msg := fmt.Sprintf("Stream Server : Default Root : Request with invalid method recieved on %s from %s", r.URL.Path, r.RemoteAddr)
		s.logMessage(msg)
		return
	}

	//404 Error Handler
	rel, err := url.Parse(s.PathPrefix)
	if err != nil {
		panic(err) //If the url parse fails, then panic.
	}

	selfPath := r.URL.ResolveReference(rel).Path

	if r.URL.Path != selfPath {
		s.writeReply(w, r, http.StatusNotFound, &jsonErrorResponse{
			Type:   "http://www.w3.org/Protocols/rfc2616/rfc2616-sec10.html",
			Detail: "The requested resource was not found. Check your URI and try again.",
			Status: http.StatusNotFound,
			Title:  http.StatusText(http.StatusNotFound),
		})
		return
	}

	resp := jsonRootResponse{}

	//Copy over metadata
	resp.Name = s.Name
	resp.Description = s.Description
	resp.GUID = s.GUID

	resp.Links = jsonhal.NewCollection()

	//Build self link
	resp.Links.Values["self"] = []*jsonhal.CollectionValue{jsonhal.CreateLink(selfPath)}

	//Build stream HATEOS links
	var streamLinks []*jsonhal.CollectionValue

	for i := 0; i < len(s.Streams); i++ {

		link := jsonhal.CreateLink(fixSlash(path.Join(s.PathPrefix, s.Streams[i].Stream.GUID, "/")))

		if s.Streams[i].Stream.Name != "" {
			link.Properties["name"] = s.Streams[i].Stream.Name
		}

		link.Properties["guid"] = s.Streams[i].Stream.GUID
		streamLinks = append(streamLinks, link)
	}
	resp.Links.Values["streams"] = streamLinks

	//Write reply to client
	s.writeReply(w, r, http.StatusOK, resp)
}

func (s *Server) writeReply(w http.ResponseWriter, r *http.Request, code int, resp interface{}) {
	err := jsonhal.WriteHalPlusJSONResp(w, code, resp)
	if err != nil {
		address := r.RemoteAddr
		path := r.URL.Path
		msg := fmt.Sprintf("RIOT Stream Server : Error occured while writing response for %s on path %s : %s", address, path, err.Error())
		s.logMessage(msg)
		panic(http.ErrAbortHandler) //Abort the HTTP transaction
	}
}

func (s *Server) logMessage(msg string) {
	if s.ErrLogger != nil {
		s.ErrLogger(msg)
	} else {
		log.Println(msg)
	}
}

func (s *Server) getRootHandler() http.HandlerFunc {
	if s.RootHandler == nil {
		return s.DefaultRoot
	}
	return s.RootHandler

}

type ctxKey string

//GetPathPrefix can be called by http handlers that sit after the RIOT server to get the path
//RIOT path prefix.
func GetPathPrefix(r *http.Request) string {
	s := getServerFromRequest(r)
	if s == nil {
		return ""
	}

	return s.PathPrefix
}

func getServerFromRequest(r *http.Request) *Server {
	s, ok := (r.Context().Value(ctxKey("Server"))).(*Server)
	if !ok {
		log.Println("Stream Server : Warning : Invalid Use of getServerFromRequest outside of stream server context")
		debug.PrintStack()
		return nil
	}
	return s
}

//LogErrorMessage logs an error message to stdout. This can only be called by handlers
//invoked after the RIOT server in an http handler chain.
func LogErrorMessage(r *http.Request, msg string) {
	s := getServerFromRequest(r)
	if s == nil {
		log.Println(msg)
	}

	s.logMessage(msg)
}

func fixSlash(p string) string {
	if len(p) == 0 {
		return "/"
	}

	if p[len(p)-1] != '/' {
		p = p + "/"
	}

	return p
}
