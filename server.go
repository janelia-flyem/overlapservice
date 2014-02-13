package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/sigu-399/gojsonschema"
        "math"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"syscall"
)

const (
	// Contain URI location for interface
	interfacePath = "/interface/"
)

// String representing interface for adder example
const ramlInterface = `#%%RAML 0.8
title: Overlap Service
/:
  post:
    description: "Call service to calculate overlap (shared voxel faces) between a set of bodies"
    body:
      application/json:
        schema: |
          { "$schema": "http://json-schema.org/schema#",
            "title": "Provide body ids whose overlap will be calculated",
            "type": "object",
            "properties": {
              "dvid-server": { 
                "description": "location of DVID server (will try to find on service proxy if not provided)",
                "type": "string" 
              },
              "uuid": { "type": "string" },
              "bodies": { 
                "description": "Array of body ids",
                "type": "array",
                "minItems": 2,
                "items": {"type": "integer", "minimum": 1},
                "uniqueItems": true
              }
            },
            "required" : ["uuid", "bodies"]
          }
    responses:
      200:
        body:
          application/json:
            schema: |
              { "$schema": "http://json-schema.org/schema#",
                "title": "Provides the size of the overlap between bodies (only reports overlap > 0)",
                "type": "object",
                "properties": {
                  "overlap-list": {
                    "description" : "List of body pairs and their overlap (body 1, body 2, overlap)",
                    "type": "array",
                    "minItems": 0,
                    "items": {
                      "type": "array",
                      "minItems": 3,
                      "maxItems": 3,
                      "items": {"type": "integer", "minimum": 1}
                    }
                  },
                "required" : ["overlap-list"]
                }
              }
/interface/interface.raml:
  get:
    description: "Get the interface for the overlap service"
    responses:
      200:
        body:
          application/raml+yaml:
`

// String representing the JSON schema for the service call
const serviceSchema = `
{ "$schema": "http://json-schema.org/schema#",
  "title": "Provide body ids whose overlap will be calculated",
  "type": "object",
  "properties": {
    "dvid-server": { 
      "description": "location of DVID server (will try to find on service proxy if not provided)",
      "type": "string" 
    },
    "uuid": { "type" : "string" },
    "bodies": { 
      "description": "Array of body ids",
      "type": "array",
      "minItems": 2,
      "items": {"type": "integer", "minimum": 1},
      "uniqueItems": true
    }
  },
  "required" : ["uuid", "bodies"]
}
`

// Address for proxy server
var proxyServer string

// webAddress is the http address for the server
var webAddress string

type overlapList [][]uint32

func (slice overlapList) Len() int {
	return len(slice)
}

func (slice overlapList) Less(i, j int) bool {
	return slice[i][2] < slice[j][2]
}

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

type sparseBodies []sparseBody

func (slice sparseBodies) Len() int {
	return len(slice)
}

func (slice sparseBodies) Less(i, j int) bool {
	return len(slice[i].rle) < len(slice[j].rle)
}

func (slice sparseBodies) Swap(i, j int) {
	slice[i], slice[j] = slice[j], slice[i]
}

type yzPair struct {
	y int32
	z int32
}

// bodyPair contains two bodies, smallest id first
type bodyPair struct {
	body1 uint32
	body2 uint32
}

func newBodyPair(body1, body2 uint32) *bodyPair {
	if body2 < body1 {
		body1, body2 = body2, body1
	}
	return &bodyPair{body1, body2}
}

type xIndex struct {
	bodyID uint32
	x      int32
	length int32
}

type xIndices []xIndex

func (slice xIndices) Len() int {
	return len(slice)
}

func (slice xIndices) Less(i, j int) bool {
	return slice[i].x < slice[j].x
}

func (slice xIndices) Swap(i, j int) {
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

// findLowerBound returns the array position to the element less than or equal to xval
func findLowerBound(xval int32, xindices xIndices) (index int, found bool) {
	index = sort.Search(len(xindices), func(i int) bool { return xindices[len(xindices)-i-1].x <= xval })
	if index == len(xindices) {
		found = false
	} else {
		found = true
		index = len(xindices) - index - 1
	}

	return
}

func findEqual(xval int32, xindices xIndices) (index int, found bool) {
	index, found = findLowerBound(xval, xindices)
	if found {
		if xindices[index].x != xval {
			found = false
		}
	}

	return
}

// computeOverlap finds the overlap between the list of bodies using the RLE, only bodies with overlap are returned
func computeOverlap(sparse_bodies sparseBodies) overlapList {
	// smallest rle first -- more memory use (or largest first for more computation)
	sort.Sort(sparse_bodies)

	// each pair of body will have 2x the overlap except for the first and last body
	var first_sparse_body = sparse_bodies[0]
	var last_sparse_body = sparse_bodies[len(sparse_bodies)-1]

	// hash of yz value to sorted slice of xIndices
	var yzmaplist = make(map[yzPair]xIndices)

	// preprocess rles -- do not load the first body
	for _, sparse_body := range sparse_bodies[1:] {
		// slice of x's
		xindices := xIndices{}
		ycurr := int32(-100000000)
		zcurr := int32(-100000000)
		for _, chunk := range sparse_body.rle {
			y := chunk.y
			z := chunk.z
			yzpair := yzPair{y, z}

			if y != ycurr || z != zcurr {
				if len(xindices) > 0 {
					yzpairold := yzPair{ycurr, zcurr}
					yzmaplist[yzpairold] = append(yzmaplist[yzpairold], xindices...)
				}

				ycurr = y
				zcurr = z
				if _, found := yzmaplist[yzpair]; !found {
					yzmaplist[yzpair] = xIndices{}
				}
				xindices = xIndices{}

			}
			xindices = append(xindices, xIndex{sparse_body.bodyID, chunk.x, chunk.length})
		}
		if len(xindices) > 0 {
			yzpairold := yzPair{ycurr, zcurr}
			yzmaplist[yzpairold] = append(yzmaplist[yzpairold], xindices...)
		}
	}

	// sort all xindices
	for _, xindices := range yzmaplist {
		sort.Sort(xindices)
	}

	// contains overlap results for each body pair
	body_pairs := make(map[bodyPair]uint32)

	// iterate one body at a time to calculate overlap, do not need to examine the last body
	for _, sparse_body := range sparse_bodies[0 : len(sparse_bodies)-1] {
		bodyid1 := sparse_body.bodyID
		for _, chunk := range sparse_body.rle {
			y := chunk.y
			z := chunk.z
			xmin := chunk.x
			xmax := xmin + chunk.length

			// examine adjacencies
			if xlist, found := yzmaplist[yzPair{y + 1, z}]; found {
				overlap(body_pairs, xlist, xmin, xmax, bodyid1)
			}
			if xlist, found := yzmaplist[yzPair{y - 1, z}]; found {
				overlap(body_pairs, xlist, xmin, xmax, bodyid1)
			}
			if xlist, found := yzmaplist[yzPair{y, z + 1}]; found {
				overlap(body_pairs, xlist, xmin, xmax, bodyid1)
			}
			if xlist, found := yzmaplist[yzPair{y, z - 1}]; found {
				overlap(body_pairs, xlist, xmin, xmax, bodyid1)
			}

			if xlist, found := yzmaplist[yzPair{y, z}]; found {
				// check if there is a pixel with a smaller x
                                // the pixel could be of the same body so check
				if index, found := findLowerBound(xmin-1, xlist); found {
                                	xval := xlist[index]
					if (bodyid1 != xval.bodyID) && (xval.length + xval.x - 1) == (xmin - 1) {
						body_pairs[*newBodyPair(bodyid1, xval.bodyID)] += 1
					}
				}

				// check if there is a pixel greater in x
                                // the pixel could be of the same body so check
				if index, found := findEqual(xmax, xlist); found {
                                    	if bodyid1 != xlist[index].bodyID { 
                                                body_pairs[*newBodyPair(bodyid1, xlist[index].bodyID)] += 1
                                        }
				}
			}
		}
	}

	for pair, val := range body_pairs {
		if pair.body1 != first_sparse_body.bodyID && pair.body1 != last_sparse_body.bodyID && pair.body2 != first_sparse_body.bodyID && pair.body2 != last_sparse_body.bodyID {
			body_pairs[pair] = val / 2
		}
	}

	overlap_slice := overlapList{}
	for pair, val := range body_pairs {
		tempslice := []uint32{pair.body1, pair.body2, val}
		overlap_slice = append(overlap_slice, tempslice)
	}

	// put body pairs with the largest overlap first
	sort.Sort(sort.Reverse(overlap_slice)) // by size of overlap

	return overlap_slice
}

// overlap calculates the overlap between bodyid1 and different bodies and puts the value in body_pairs
func overlap(body_pairs map[bodyPair]uint32, xlist xIndices, xmin int32, xmax int32, bodyid1 uint32) {
	var maxindex int
        var minindex int
        var found bool
	// grab the last index less than or equal to the largest index in the body
        if maxindex, found = findLowerBound(xmax-1, xlist); !found {
		return
	}

	// get lower bound from min
	if minindex, found = findLowerBound(xmin-1, xlist); !found {
		minindex = 0
	}

	for i := minindex; i <= maxindex; i += 1 {
		if xlist[i].bodyID != bodyid1 {
			length := xlist[i].length
			start := xlist[i].x
			if start < xmin {
				length -= (xmin - start)
				start = xmin
			}

			if length > 0 {
				body_pairs[*(newBodyPair(bodyid1, xlist[i].bodyID))] += uint32(math.Min(float64(length), float64(xmax-start)))
			}
		}
	}
}

// serviceHandler handlers post request to "/jobs"
func serviceHandler(w http.ResponseWriter, r *http.Request) {
	pathlist, requestType, err := parseURI(r, "/")
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
	http.HandleFunc("/", serviceHandler)

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
