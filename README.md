ThinkSeeDo Go Packages
======================

Custom Go packages. Command line tools are in the `cmds` directory.

### Install Pipeline Viewer

All command and package sources include any generated sources. So, `go get`
works right away.

    go get github.com/stephen-soltesz/go/cmds/lineprobe
    go get github.com/stephen-soltesz/go/cmds/lineviewer

### Dependencies

To recreate the generated sources, install these dependencies.

GopherJS - translates syntacically correct go code to javascript.

    go get github.com/gopherjs/gopherjs

go-bindata - converts binary assets into go code to be bundled with go binaries.

    go get github.com/jteeuwen/go-bindata
