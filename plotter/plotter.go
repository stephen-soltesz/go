/*
Package plotter abstracts the details of other chart packages into a simple
interface composed of Figures, Charts, and Data
*/
package plotter

import (
	"fmt"
	//"image"
	"image/color"
	//"image/png"
	"io"
	"os"
	"path"
	"time"

	// third-party
	//"code.google.com/p/freetype-go/freetype"
	//"code.google.com/p/freetype-go/freetype/truetype"
	//"github.com/vdobler/chart"
	//"github.com/vdobler/chart/imgg"
	//"github.com/vdobler/chart/svgg"
	"github.com/ajstarks/svgo"
	"github.com/stephen-soltesz/chart"
	//"github.com/stephen-soltesz/chart/imgg"
	"github.com/stephen-soltesz/chart/svgg"
)

type Figure interface {
	Render(writer io.Writer, width, height int) error
	AddChart(title, xlabel, ylabel string, xmin, xmax float64) *Chart
}

type implFigure struct {
	Charts   []*Chart
	fontName string
	usetime  bool
}

type Style chart.Style

type Chart struct {
	chart.ScatterChart
}

type Data struct {
	chart.ScatterChartData
}

func autoStyle(s int) func() Style {
	i := s
	l := len(chart.StandardColors)
	return func() Style {
		i = (i + 1) % l
		return Style{Symbol: '.',
			SymbolSize:  1,
			SymbolColor: chart.StandardColors[i],
			LineColor:   chart.StandardColors[i],
			LineStyle:   chart.SolidLine,
			LineWidth:   2,
			FillColor:   chart.StandardColors[i]}
	}
}

var NextStyle = autoStyle(0)

func NewFigure(usetime bool) Figure {
	f := implFigure{}
	f.fontName = "FreeSans.ttf"
	f.usetime = usetime
	return &f
}

func RenderToFile(f Figure, filename string, width, height int) error {
	var err error
	var imgFile *os.File
	dir, file := path.Split(filename)
	tmpname := dir + "/.tmp_" + file

	imgFile, err = os.Create(tmpname)
	if err != nil {
		return err
	}
	err = f.Render(imgFile, width, height)
	if err != nil {
		return err
	}
	err = imgFile.Close()
	if err != nil {
		return err
	}
	err = os.Rename(tmpname, filename)
	if err != nil {
		return err
	}
	return nil
}

func (f *implFigure) Render(writer io.Writer, width, height int) error {
	whiteBG := color.RGBA{0xee, 0xee, 0xee, 0xff}
	count := len(f.Charts)

	svgdata := svg.New(writer)
	svgdata.Start(width+10, height+10)
	svgdata.Rect(0, 0, width, height, "fill: #ffffff")

	for i, ax := range f.Charts {
		igr := svgg.AddTo(svgdata, 0, i*(height/count), width, (height / count), "", 12, whiteBG)
		ax.Plot(igr)
	}

	svgdata.End()
	return nil
}

func (f *implFigure) AddChart(title, xlabel, ylabel string, xmin, xmax float64) *Chart {
	axis := Chart{chart.ScatterChart{}}

	axis.Title = title
	axis.XRange.Label = xlabel
	axis.YRange.Label = ylabel
	axis.XRange.ShowZero = false
	axis.XRange.TicSetting.Mirror = chart.MirrorAxisOnly
	axis.YRange.TicSetting.Mirror = chart.MirrorAxisOnly
	axis.XRange.TicSetting.Grid = chart.GridLines
	axis.YRange.TicSetting.Grid = chart.GridLines
	axis.YRange.TicSetting.Format = func(val float64) string {
		return fmt.Sprintf("%.1f", val)
	}
	axis.XRange.Time = f.usetime
	if f.usetime {
		axis.XRange.TFixed(time.Unix(int64(xmin), 0), time.Unix(int64(xmax), 0), nil)
	} else {
		axis.XRange.Fixed(xmin, xmax, 0)
	}
	axis.XRange.Init()

	fSmall := chart.Font{Size: chart.SmallFontSize}
	fNormal := chart.Font{Size: chart.NormalFontSize}
	sKey := chart.Style{LineColor: color.Gray{0x88},
		LineStyle: chart.SolidLine,
		LineWidth: 1,
		FillColor: color.NRGBA{0xf8, 0xf8, 0xf8, 0x66},
		Font:      fSmall}
	sGrid := chart.Style{LineStyle: chart.DashedLine,
		LineColor: color.Gray{0xbb},
		LineWidth: 1}
	sZero := chart.Style{LineStyle: chart.DashedLine,
		LineColor: color.Gray{0xbb}}
	sMajor := chart.Style{LineColor: color.Gray{0x88},
		LineWidth: 1,
		Font:      fNormal}
	sTic := chart.Style{LineWidth: 1,
		LineColor: color.Gray{0x88},
		Font:      fNormal}
	sRange := chart.Style{LineWidth: 1, Font: fNormal}
	sTitle := chart.Style{LineWidth: 1, Font: fNormal}

	//sBg := chart.Style{LineColor: color.NRGBA{0xff, 0xff, 0xee, 0x88},
	//                   FillColor: color.NRGBA{0xff, 0xff, 0xee, 0x88}}
	//                   chart.PlotBackgroundElement: sBg,

	axis.Options = chart.PlotOptions{
		chart.GridLineElement:   sGrid,
		chart.ZeroAxisElement:   sZero,
		chart.MajorAxisElement:  sMajor,
		chart.MinorAxisElement:  sMajor,
		chart.MajorTicElement:   sTic,
		chart.MinorTicElement:   sTic,
		chart.KeyElement:        sKey,
		chart.RangeLimitElement: sRange,
		chart.TitleElement:      sTitle}
	axis.Key.Cols = 1
	axis.Key.Pos = "itl"

	f.Charts = append(f.Charts, &axis)
	return &axis
}

func (ax *Chart) ShowXAxis(s bool) {
	ax.XRange.TicSetting.HideLabels = !s
	return
}

func (ax *Chart) AddData(name string, x, y []float64, style Style) *Data {
	ax.AddDataPair(name, x, y, chart.PlotStyleLinesPoints, chart.Style(style))
	line := Data{ax.Data[len(ax.Data)-1]}
	return &line
}

/*
// must be called after a plot
func (ax *Chart) PlotArea() (int, int, int, int) {
  x1 := ax.XRange.Data2Screen(ax.XRange.Min)
  y1 := ax.YRange.Data2Screen(ax.YRange.Min)
  w1 := ax.XRange.Data2Screen(ax.XRange.Max) - ax.XRange.Data2Screen(ax.XRange.Min)
  h1 := ax.YRange.Data2Screen(ax.YRange.Max) - ax.YRange.Data2Screen(ax.YRange.Min)
  return x1, y1, w1, h1
}
*/
/*
func (f *implFigure) RenderPNG(writer io.Writer, width, height int) error {
	whiteBG := color.RGBA{0xee, 0xee, 0xee, 0xff}
	image := image.NewRGBA(image.Rect(0, 0, width, height))
	count := len(f.Charts)
	font := loadFont(f.fontName)

	for i, ax := range f.Charts {
		igr := imgg.AddTo(image, 0, i*(height/count), width, (height / count), whiteBG, font, nil)
		ax.Plot(igr)
	}

	return png.Encode(writer, image)
}

var fontCache map[string]*truetype.Font

func loadFont(fontfile string) *truetype.Font {
	var font *truetype.Font
	var ok bool
	var err error
	var data []byte

	// Read the font data once.
	if fontCache == nil {
		fontCache = make(map[string]*truetype.Font)
	}
	fontpath := "res/" + fontfile
	if font, ok = fontCache[fontpath]; !ok {
		data, err = Asset(fontpath)
		if err != nil {
			return nil
		}
		font, err = freetype.ParseFont(data)
		if err != nil {
			return nil
		}
		fontCache[fontpath] = font
	}
	return font
}
*/
