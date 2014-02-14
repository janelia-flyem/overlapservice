package overlap

// String representing the JSON schema for the service call
const overlapSchema = `
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
      "description": "Array of body ids (should be unsigned ints but for some reason validator requries a number type",
      "type": "array",
      "minItems": 2,
      "items": {"type": "number", "minimum": 1},
      "uniqueItems": true
    }
  },
  "required" : ["uuid", "bodies"]
}
`

const statsSchema = `
{ "$schema": "http://json-schema.org/schema#",
  "title": "Provide body ids whose stats will be calculated",
  "type": "object",
  "properties": {
    "dvid-server": { 
      "description": "location of DVID server (will try to find on service proxy if not provided)",
      "type": "string" 
    },
    "uuid": { "type" : "string" },
    "bodies": { 
      "description": "Array of body ids (should be unsigned ints but for some reason validator requries a number type",
      "type": "array",
      "minItems": 1,
      "items": {"type": "number", "minimum": 1},
      "uniqueItems": true
    }
  },
  "required" : ["uuid", "bodies"]
}
`
