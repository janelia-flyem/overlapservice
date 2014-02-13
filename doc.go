/*
overlapservice is a simple REST service written in go
that calculates the overlap between a set of bodies, as
defined/stored in DVID.  Currently, the contact area is determined by 6-connectivity and is a slight
over-estimate of surface area as it actually returns the number of voxel faces
that touch each other between a set of bodies.

To launch the service:

    % overlapservice [-proxy PROXYADDRESS (default "")] [-port WEBPORT (default 15123)] [-registry REGISTRYADDRESS (default "")]

This will start a web server at the given port on the current
machine (ADDR).  Optional: The registry address specifies the serviceproxy registry location
(e.g., "127.0.0.1:7946" if serviceproxy was launched at this address).  The proxy
address is the location of the serviceproxy http server (e.g. "127.0.0.1:15333" if
serviceproxy web server was launched at this address).

The simplest way to use the server is navigate to "http://ADDR" and submit the provided form.
A DVID server location, UUID from DVID, and a set of body IDs must be provided.  Optionally, one can post
a JSON directly to the service at URI /service.  Below is a sample JSON:

{
    "dvid-server" : "blah.com:12345",
    "uuid": "4234",
    "bodies": [100, 140, 233]
}

After posting this data, overlap (in terms of the number of touching voxel faces) will be returned
for each pair).  Pairs without overlap will not be returned.

For more details, the rest interface specification is in RAML (http://raml.org) format.
To view the interface, navigate to "http://ADDR/interface". 
*/
package main
