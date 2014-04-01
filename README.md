ThinkSeeDo Go Packages
======================

Go packages and command line tools are in the `cmds` directory.

### Dependencies

GopherJS:

    go get github.com/gopherjs/gopherjs

go-bindata:

    go get github.com/jteeuwen/go-bindata


### Build

    go get github.com/stephen-soltesz/go/cmds/lineviewer

You may need to manually, run 'make', to generate the gopherjs and bindata
dependencies for the lineviewer.

    cd $GOPATH/src/github.com/stephen-soltesz/go/cmds/lineviewer
    make
