ThinkSeeDo Go Packages
======================

Go packages and command line tools are in the `cmds` directory.

Depends on
==========

GopherJS:

    go get github.com/gopherjs/gopherjs

go-bindata:

		go get github.com/jteeuwen/go-bindata


Finally:

    go get thinkseedo.com/go/cmds/lineviewer

You may need to manually, run 'make', to generate the gopherjs and bindata
dependencies for the lineviewer.

    cd $GOPATH/src/thinkseedo.com/go/cmds/lineviewer
		make
