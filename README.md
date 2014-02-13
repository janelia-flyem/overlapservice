# Service Proxy Tool

[![Build Status](https://drone.io/github.com/janelia-flyem/overlapservice/status.png)](https://drone.io/github.com/janelia-flyem/overlapservice/latest)


This go package calculates the overlap between a set of bodies, as
defined/stored in [DVID](https://github.com/janelia-flyem/dvid).  For
example, if there are two segmented neurons in a dataset stored in DVID,
this service can be used to find the contact area between those neurons.
Currently, the contact area is determined by 6-connectivity and is a slight
over-estimate of surface area as it actually returns the number of voxel faces
that touch each other between a set of bodies.

The tool has been tested on linux but should work in other environments.
It also works in service-oriented environment.  It can register itself
via [serviceproxy](https://github.com/janelia-flyem/serviceproxy) and automatically
look for the existence of DVID server via the proxy.


##Installation and Basic Usage
This package includes the main executable for launching the
overlap service.

To install overlapservice:

    % go get github.com/janelia-flyem/overlapservice

To launch the service:

    % overlapservice [-port WEBPORT (default 15333)]

This will start a web server at the given port on the current
machine.

The rest interface specification is in [RAML](http://raml.org) format.
To view the interface, navigate to
"127.0.0.1:WEBPORT/interface".  To view the RAML interface, we use
the [api-console](https://github.com/mulesoft/api-console) javascript-based viewer.


