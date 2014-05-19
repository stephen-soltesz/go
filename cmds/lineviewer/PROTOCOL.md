## LineViewer Protocol Specification

The lineviewer consists of two parts:
 * the collector, which receives string-format data from clients.
 * the viewer, which responds to browser or json clients via http.

The details for each component are outlined below.

### Collector

The collector server listens for new client connections on PORT (3131 by
default). Every connection adds a new line. The server treats the connection as
read-only, and the client should treat the connection as write-only.  The
client can specify style hints for the axis and line before sending any plot
data. If unspecified, the server assigns sensible default values for the line
color, line label, and axis. 

All data sent by the client should match this approximate BNF pattern:

  <all-data> ::= <directives> <value-stream>
  <directives> ::= <style-set> | <command> <EOL>
  <command> ::= exit | reset
  <style-set> ::= <style-set> <style-element> <EOL> 
  <style-element> ::= "axis:" <axis-name> ":" <x-label> ":" <y-label> |
	                    "axis-scale:" <x-scale> ":" <y-scale> |
	                    "axis-ylimit:" <ymin> ":" <ymax> |
	                    "axis-xlimit:" <xmin> ":" <xmax> |
	                    "line-label:" <line-name> |
	                    "line-color:" <line-color>
  <x-scale> ::= <ignored>
  <y-scale> ::= log | linear
	<line-color> ::= "#" [0-f]{6}

  <value-stream> ::= <value-stream> <value>
  <value> ::= <float> <EOL> | <EOF>

After the server receives EOF, the connection has closed, and this terminates
the line.

### Viewer

The viewer by default runs on port 8080. It is a simple http server that
exposes the following urls:

 * / - root page, returns user interface for interactive viewer.
 * /config - returns a json object with the viewer configuration, such as
   plotHeight, plotWidth.
 * /svg - returns an svg document that represents the current state of the collector.
