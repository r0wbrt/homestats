//Copyright Robert C. Taylor 2018
//Distributed under the terms of the LICENSE file

package jsonhal

import (
	"encoding/json"
	"net/http"
)

//WriteHalPlusJSONResp returns a hal+json response via http. Sets the headers of the response
//appropriately.
func WriteHalPlusJSONResp(w http.ResponseWriter, code int, obj interface{}) error {

	data, err := json.Marshal(obj)
	if err != nil {
		return err
	}

	w.Header().Add("Content-Type", "application/hal+json")
	w.WriteHeader(code)
	w.Write(data)

	return nil
}
