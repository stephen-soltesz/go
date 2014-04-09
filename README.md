ThinkSeeDo Go Packages
======================

Custom Go packages. Command line tools are in the `cmds` directory.

### Install Pipeline Viewer

All command and package sources include any generated sources. So, `go get`
works right away.

    go get github.com/stephen-soltesz/go/cmds/lineprobe
    go get github.com/stephen-soltesz/go/cmds/lineviewer

Together, these two commands make it easy to view streams of data in real time
from the command line.

Lineviewer combines two servers: one to receive data sent by lineprobe and a
second to plot and display data over http. Multiple lineprobe clients can
connect to a single lineviewer server to plot multiple lines and axes.

### Dependencies

To recreate the generated sources, install these dependencies.

GopherJS - translate Go code to javascript.

    go get github.com/gopherjs/gopherjs

go-bindata - convert binary assets to Go code to be bundled with Go binaries.

    go get github.com/jteeuwen/go-bindata

### Example

![Pipeline Example](https://github.com/stephen-soltesz/go/raw/master/screenshots/example.png)

First, start the server:

    lineviewer --timestamp

Then run one or more lineprobes.

    lineprobe --label "process count" --command "ps ax | wc -l"

And, finally open a browser to:

    http://localhost:8080/

See the command help for more examples and options.
