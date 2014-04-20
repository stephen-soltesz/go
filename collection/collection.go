/*
Collection is a container class for organizing raw line coordinate data.

Collection containers are hierarchical.  A Collection contains one or more Axis
objects. And each Axis contains one or more Line objects. So, if you only need
a single "Collection", a default is provided using:

    collection.Default().

Since collections exist to help with plotting line data, there is one
convenience function for plotting. To generate an image representing the
collection data, use:

		collection.Plot()
*/
package collection

import (
	"encoding/hex"
	"image/color"
	"io"
	"math"
	"time"

	// third party
	"github.com/stephen-soltesz/go/plotter"
)

var defaultCollection *Collection

// A Collection is the root container.
// Axes is a map, so that Axis objects can be referred to by name, see GetAxis().
type Collection struct {
	Title   string
	Axes    map[string]*Axis
	Usetime bool
}

// A Collection contains one or more Axis objects.
type Axis struct {
	Name   string
	XLabel string
	YLabel string
	Lines  map[string]*Line
	Ylimit bool
	Ymin, Ymax	 float64
	Uselog  bool
}

// An Axis contains one or more Line objects.
// The X and Y arrays contain coordinates of each point on the Line.
type Line struct {
	Name  string
	Style plotter.Style
	X     []float64
	Y     []float64

	// internal book keeping
	xyrange
}

type xyrange struct {
	// internal book keeping
	max_x float64
	min_x float64
	max_y float64
	min_y float64
}

func initXYrange() xyrange {
	return xyrange{math.SmallestNonzeroFloat64, math.MaxFloat64,
		math.SmallestNonzeroFloat64, math.MaxFloat64}
}

func (xy *xyrange) updateXYrange(x, y float64) {
	xy.min_x = math.Min(x, xy.min_x)
	xy.max_x = math.Max(x, xy.max_x)

	xy.min_y = math.Min(y, xy.min_y)
	xy.max_y = math.Max(y, xy.max_y)
}

func New() *Collection {
	c := &Collection{}
	c.Axes = make(map[string]*Axis)
	return c
}

func Default() *Collection {
	if defaultCollection == nil {
		defaultCollection = New()
	}
	return defaultCollection
}

// Returns the Axis associated with given name. If no Axis exists, a new Axis
// is created and associated with given name.
func (c *Collection) GetAxis(name string) *Axis {
	// TODO: make thread safe
	if axis, ok := c.Axes[name]; !ok {
		return c.AddAxis(name, "", "")
	} else {
		return axis
	}
}

func (c *Collection) AddAxis(name, xlabel, ylabel string) *Axis {
	axis := &Axis{name, xlabel, ylabel, make(map[string]*Line), false, 0, 0, false}
	c.Axes[name] = axis
	return axis
}

func (ax *Axis) GetLine(name string) *Line {
	// TODO: make thread safe
	if line, ok := ax.Lines[name]; !ok {
		return ax.AddLine(name)
	} else {
		return line
	}
}

func (ax *Axis) AddLine(name string) *Line {
	line := &Line{}
	line.Name = name
	line.Style = plotter.NextStyle()
	line.xyrange = initXYrange()
	ax.Lines[name] = line
	return line
}

func (line *Line) SetColor(hexColor string) {
	if len(hexColor) > 0 && hexColor[0] == '#' {
		hexColor = hexColor[1:]
	}
	if len(hexColor) == 6 {
		rgb, err := hex.DecodeString(hexColor)
		if err != nil {
			return
		}
		r, g, b := rgb[0], rgb[1], rgb[2]
		line.Style.SymbolColor = color.RGBA{r, g, b, 255}
		line.Style.LineColor = color.RGBA{r, g, b, 255}
		line.Style.FillColor = color.RGBA{r, g, b, 255}
	}
}

// Append adds a new x,y coordinate to the end of the Line.
func (line *Line) Append(x, y float64) {
	line.X = append(line.X, x)
	line.Y = append(line.Y, y)
	line.xyrange.updateXYrange(x, y)
}

// Count returns the length of Line X and Y coordinates.
func (line *Line) Count() (int, int) {
	return len(line.X), len(line.Y)
}

func (line *Line) RangeX() (float64, float64) {
	min := line.xyrange.min_x - 0.1
	max := line.xyrange.max_x + 0.1
	return min, max
}

func (line *Line) RangeY() (float64, float64) {
	min := line.xyrange.min_y - 0.1
	max := line.xyrange.max_y + 0.1
	return min, max
}

// MaxX returns the greatest X coordinate for all Lines in this Axis.
func (ax *Axis) MaxX() float64 {
	xmax := 0.0
	if len(ax.Lines) == 0 {
		return xmax
	}
	for _, line := range ax.Lines {
		xlen, _ := line.Count()
		if xlen == 0 {
			continue
		}
		_, lmax := line.RangeX()
		xmax = math.Max(lmax, xmax)
	}
	return xmax
}

func (c *Collection) Plot(writer io.Writer, width, height int, offset float64, samples float64) error {

	fig := plotter.NewFigure(c.Usetime)
	for _, ax := range c.Axes {
		xmax := 0.0
		if c.Usetime {
			// continuously update plot with most current time.
			// TODO: make this an option.
			xmax = float64(time.Now().Unix())-offset
		} else {
			xmax = ax.MaxX()
		}
		chart := fig.AddChart(c.Title, ax.XLabel, ax.YLabel, xmax-samples, xmax)
		chart.YRange.Log = ax.Uselog
		if ax.Ylimit {
			chart.YRange.Fixed(ax.Ymin, ax.Ymax, 0)
			chart.YRange.Init()
		}
		for _, line := range ax.Lines {
			if xc, _ := line.Count(); xc > 0 {
				chart.AddData(line.Name, line.X, line.Y, line.Style)
			}
		}
	}

	return fig.RenderSVG(writer, width, height)
}
