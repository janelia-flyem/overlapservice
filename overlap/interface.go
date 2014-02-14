package overlap

// String representing interface for adder example
const ramlInterface = `#%%RAML 0.8
title: Overlap Service
/overlap:
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
/bodystats:
  post:
    description: "Call service to calculate statistics over a set of bodies"
    body:
      application/json:
        schema: |
          { "$schema": "http://json-schema.org/schema#",
            "title": "Provide body ids whose stats will be calculated",
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
                "minItems": 1,
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
                "title": "Provides the size and surface area of each body",
                "type": "object",
                "properties": {
                  "body-stats": {
                    "description" : "List of bodies with stats (body id, size, surface area)",
                    "type": "array",
                    "minItems": 0,
                    "items": {
                      "type": "array",
                      "minItems": 3,
                      "maxItems": 3,
                      "items": {"type": "integer", "minimum": 1}
                    }
                  },
                "required" : ["body-stats"]
                }
              }
/interface/interface.raml:
  get:
    description: "Get the interface for the overlap and body service"
    responses:
      200:
        body:
          application/raml+yaml:
`
