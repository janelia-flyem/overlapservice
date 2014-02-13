package overlap 

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/sigu-399/gojsonschema"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
)

const (
	// Contain URI location for interface
	interfacePath = "/interface/"
)

// Address for proxy server
var proxyServer string

// webAddress is the http address for the server
var webAddress string

// overlapList contains the final output as a slice of [body1, body2, overalap]
type overlapList [][]uint32

// Len to enable sorting by overlap amount
func (slice overlapList) Len() int {
	return len(slice)
}

// Less to enable sorting by overlap amount
func (slice overlapList) Less(i, j int) bool {
	return slice[i][2] < slice[j][2]
}

// Swap to enable sorting by overlap amount
func (slice overlapList) Swap(i, j int) {
	slice[i], slice[j] = slice[j], slice[i]
}

// sparseData encodes the run length for part of a body
type sparseData struct {
	x      int32
	y      int32
	z      int32
	length int32
}

// sparseBody provides the run length encoding of the body
type sparseBody struct {
	bodyID uint32
	rle    []sparseData
}

// sparseBodies contains a slice of rle bodies
type sparseBodies []sparseBody

// Len to enable sorting by number of spans 
func (slice sparseBodies) Len() int {
	return len(slice)
}

// Less to enable sorting by number of spans 
func (slice sparseBodies) Less(i, j int) bool {
	return len(slice[i].rle) < len(slice[j].rle)
}

// Swap to enable sorting by number of spans 
func (slice sparseBodies) Swap(i, j int) {
	slice[i], slice[j] = slice[j], slice[i]
}


// parseURI is a utility function for retrieving parts of the URI
func parseURI(r *http.Request, prefix string) ([]string, string, error) {
	requestType := strings.ToLower(r.Method)
	prefix = strings.Trim(prefix, "/")
	path := strings.Trim(r.URL.Path, "/")
	prefix_list := strings.Split(prefix, "/")
	url_list := strings.Split(path, "/")
	var path_list []string

	if len(prefix_list) > len(url_list) {
		return path_list, requestType, fmt.Errorf("Incorrectly formatted URI")
	}

	for i, val := range prefix_list {
		if val != url_list[i] {
			return path_list, requestType, fmt.Errorf("Incorrectly formatted URI")
		}
	}

	if len(prefix_list) < len(url_list) {
		path_list = url_list[len(prefix_list):]
	}

	return path_list, requestType, nil
}

// badRequest is a halper for printing an http error message
func badRequest(w http.ResponseWriter, msg string) {
	fmt.Println(msg)
	http.Error(w, msg, http.StatusBadRequest)
}

// getDVIDserver retrieves the server from the JSON or looks it up
func getDVIDserver(jsondata map[string]interface{}) (string, error) {
	if _, found := jsondata["dvid-server"]; found {
		return jsondata["dvid-server"].(string), nil
	} else if proxyServer != "" {
                resp, err := http.Get("http://" + proxyServer + "/services/dvid/node")
                if err != nil {
			return "", fmt.Errorf("dvid server not found at proxy")
			// handle error
		}
		defer resp.Body.Close()
		decoder := json.NewDecoder(resp.Body)
		dvidnode := make(map[string]interface{})
		err = decoder.Decode(&dvidnode)
		if err != nil {
			return "", fmt.Errorf("Error decoding JSON from proxy server")
		}
		if dvidnode["service-location"] == nil {
			return "", fmt.Errorf("No service location found for DVID")
		}
		return dvidnode["service-location"].(string), nil
	}
        return "", fmt.Errorf("No proxy server location exists")
}

// InterfaceHandler returns the RAML interface for any request at
// the /interface URI.
func interfaceHandler(w http.ResponseWriter, r *http.Request) {
	// allow resources to be accessed via ajax
	w.Header().Set("Content-Type", "application/raml+yaml")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	fmt.Fprintf(w, ramlInterface)
}



// serviceHandler handlers post request to "/jobs"
func serviceHandler(w http.ResponseWriter, r *http.Request) {
	pathlist, requestType, err := parseURI(r, "/service/")
	if err != nil || len(pathlist) != 0 {
		badRequest(w, "Error: incorrectly formatted request")
		return
	}
	if requestType != "post" {
		badRequest(w, "only supports posts")
		return
	}

	// read json
	decoder := json.NewDecoder(r.Body)
	var json_data map[string]interface{}
	err = decoder.Decode(&json_data)

	// convert schema to json data
	var schema_data interface{}
	json.Unmarshal([]byte(serviceSchema), &schema_data)

	// validate json schema
	schema, err := gojsonschema.NewJsonSchemaDocument(schema_data)
	validationResult := schema.Validate(json_data)
	if !validationResult.IsValid() {
		badRequest(w, "JSON did not pass validation")
		return
	}

	// retrieve dvid server
	dvidserver, err := getDVIDserver(json_data)
	if err != nil {
		badRequest(w, "DVID server could not be located on proxy")
		return
	}
                
	// get data uuid
	uuid := json_data["uuid"].(string)

	// base url for all dvid queries
	baseurl := dvidserver + "/api/node/" + uuid + "/sp2body/sparsevol/"

	// read data for each body
	sparse_bodies := sparseBodies{}

	bodyinter_list := json_data["bodies"].([]interface{})
	for _, bodyinter := range bodyinter_list {
		bodyid := int(bodyinter.(float64))
		url := baseurl + strconv.Itoa(bodyid)

		resp, err := http.Get(url)
                if err != nil || resp.StatusCode != 200 {
			badRequest(w, "Body could not be read from "+url)
			return
		}
		defer resp.Body.Close()

		// not examing initial body data for now
		var junk uint32
		binary.Read(resp.Body, binary.LittleEndian, &junk)
		binary.Read(resp.Body, binary.LittleEndian, &junk)

		var numspans uint32
		binary.Read(resp.Body, binary.LittleEndian, &numspans)

		sparse_body := sparseBody{}
                sparse_body.bodyID = uint32(bodyid)

		for iter := 0; iter < int(numspans); iter += 1 {
			var x, y, z, run int32
			err := binary.Read(resp.Body, binary.LittleEndian, &x)
			if err != nil {
				badRequest(w, "Sparse body encoding incorrect")
				return
			}
			err = binary.Read(resp.Body, binary.LittleEndian, &y)
			if err != nil {
				badRequest(w, "Sparse body encoding incorrect")
				return
			}
			err = binary.Read(resp.Body, binary.LittleEndian, &z)
			if err != nil {
				badRequest(w, "Sparse body encoding incorrect")
				return
			}
			err = binary.Read(resp.Body, binary.LittleEndian, &run)
			if err != nil {
				badRequest(w, "Sparse body encoding incorrect")
				return
			}

			sparse_data := sparseData{x, y, z, run}
                        
                        sparse_body.rle = append(sparse_body.rle, sparse_data)
		}
		sparse_bodies = append(sparse_bodies, sparse_body)
	}

	// algorithm for computing overlap -- empty if there is no overlap
	overlap_list := computeOverlap(sparse_bodies)
	json_struct := make(map[string]overlapList)
	json_struct["overlap-list"] = overlap_list

	w.Header().Set("Content-Type", "application/json")

	jsondata, _ := json.Marshal(json_struct)
	fmt.Fprintf(w, string(jsondata))
}

// Serve is the main server function call that creates http server and handlers
func Serve(proxyserver string, port int) {
	proxyServer = proxyserver

	hname, _ := os.Hostname()
	webAddress = hname + ":" + strconv.Itoa(port)

	fmt.Printf("Web server address: %s\n", webAddress)
	fmt.Printf("Running...\n")

	httpserver := &http.Server{Addr: webAddress}

	// serve out static json schema and raml (allow access)
	http.HandleFunc(interfacePath, interfaceHandler)

	// perform service
	http.HandleFunc("/service/", serviceHandler)

	// exit server if user presses Ctrl-C
	go func() {
		sigch := make(chan os.Signal)
		signal.Notify(sigch, os.Interrupt, syscall.SIGTERM)
		<-sigch
		fmt.Println("Exiting...")
		os.Exit(0)
	}()

	httpserver.ListenAndServe()
}
