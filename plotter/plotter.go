/*
Package plotter abstracts the details of other chart packages into a simple
interface composed of Figures, Charts, and Data
*/
package plotter

import (
  "image"
  "image/color"
	"image/png"
  "io"
  "os"
  "path"
	"time"

	// third-party
  "code.google.com/p/freetype-go/freetype"
  "code.google.com/p/freetype-go/freetype/truetype"
  //"github.com/vdobler/chart"
	//"github.com/vdobler/chart/imgg"
  "github.com/stephen-soltesz/chart"
	"github.com/stephen-soltesz/chart/imgg"
)

type Style chart.Style

type Figure struct {
  Charts []*Chart
	fontName string
}

type Chart struct {
	chart.ScatterChart
}

type Data struct {
  chart.ScatterChartData
}

func autoStyle(s int) (func() Style) {
  i := s
  l := len(chart.StandardColors)
  return func() Style {
    i = (i + 1) % l
    return Style{Symbol: '.',
                       SymbolSize: 1,
                       SymbolColor: chart.StandardColors[i],
                       LineColor: chart.StandardColors[i],
                       LineStyle: chart.SolidLine,
                       LineWidth: 2,
                       FillColor: chart.StandardColors[i]}
  }
}

var NextStyle = autoStyle(0)

func loadFont(fontfile string) *truetype.Font {
  // Read the font data.
	data, err := Asset("res/"+fontfile)
	if err != nil {
    return nil
	}
  font, err := freetype.ParseFont(data)
  if err != nil {
    return nil
  }
  return font
}

func NewFigure() *Figure {
  f := Figure{}
	f.fontName = "FreeSans.ttf"
  return &f
}

func (f *Figure) RenderFile(filename string, width, height int) error {
	var err error
  var imgFile *os.File
  dir, file := path.Split(filename)
  tmpname := dir+"/.tmp_"+file

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

func (f *Figure) Render(writer io.Writer, width, height int) error {
  whiteBG := color.RGBA{0xee, 0xee, 0xee, 0xff}
	image := image.NewRGBA(image.Rect(0, 0, width, height))
  count := len(f.Charts)
  font := loadFont(f.fontName)

  for i, ax := range f.Charts {
	  igr := imgg.AddTo(image, 0, i*(height/count), width, (height/count), whiteBG, font, nil)
	  ax.Plot(igr)
  }

	return png.Encode(writer, image)
}

func (f *Figure) AddChart(title, xlabel, ylabel string, xmin, xmax float64, usetime bool) *Chart {
	axis := Chart{chart.ScatterChart{}}

	axis.Title = title
	axis.XRange.Label = xlabel
  axis.YRange.Label = ylabel
	axis.XRange.ShowZero = false
	axis.XRange.TicSetting.Mirror = chart.MirrorAxisOnly
	axis.YRange.TicSetting.Mirror = chart.MirrorAxisOnly
	axis.XRange.TicSetting.Grid = chart.GridLines
	axis.YRange.TicSetting.Grid = chart.GridLines
	axis.XRange.Time = usetime
	if usetime {
		axis.XRange.TFixed(time.Unix(int64(xmin),0), time.Unix(int64(xmax),0), nil)
		// TODO make configurable
		//axis.YRange.Log = true
	} else {
		axis.XRange.Fixed(xmin, xmax, 0)
	}
	axis.XRange.Init()

  fSmall := chart.Font{Size: chart.NormalFontSize}
  //sKey := chart.Style{LineColor: color.NRGBA{0x0f, 0x0f, 0x0f, 0xff},
  //                    LineStyle: chart.SolidLine, LineWidth: 1,
  //                    FillColor: color.NRGBA{0xf8, 0xf8, 0xf8, 0xff},
  //                    Font: chart.Font{Size: chart.NormalFontSize}}

  sGrid := chart.Style{LineStyle: chart.DottedLine, LineColor: color.Gray{0xcc}}
  sZero := chart.Style{LineStyle: chart.DottedLine, LineColor: color.Gray{0xcc}}
  sMajor := chart.Style{LineColor: color.Gray{0x88}, Font: fSmall}
  sTic := chart.Style{LineColor: color.Gray{0x88}, Font: fSmall}
  sRange := chart.Style{Font: fSmall}
  //sBg := chart.Style{LineColor: color.NRGBA{0xff, 0xff, 0xee, 0x88},
  //                   FillColor: color.NRGBA{0xff, 0xff, 0xee, 0x88}}
  sTitle := chart.Style{Font: chart.Font{"Arial", chart.NormalFontSize, color.Gray{0x88}}}

  axis.Options = chart.PlotOptions{chart.GridLineElement: sGrid,
   //                                  chart.PlotBackgroundElement: sBg,
                                     chart.ZeroAxisElement: sZero,
                                     chart.MajorAxisElement: sMajor,
                                     chart.MinorAxisElement: sMajor,
                                     chart.MajorTicElement: sTic,
                                     chart.MinorTicElement: sTic,
                                     //chart.KeyElement: sKey,
                                     chart.RangeLimitElement: sRange,
                                     chart.TitleElement: sTitle}
	axis.Key.Cols = 1
	axis.Key.Pos = "itl"

	f.Charts = append(f.Charts, &axis)
  return &axis
}

func (ax *Chart) ShowXAxis (s bool) {
  ax.XRange.TicSetting.HideLabels = !s
  return
}

func (ax *Chart) AddData(name string, x, y []float64, style Style) *Data {
	ax.AddDataPair(name, x, y, chart.PlotStyleLinesPoints, chart.Style(style))
  line := Data{ax.Data[len(ax.Data)-1]}
  return &line
}

//func (ax *Chart) LenDatas() int {
//  return len(ax.Data)
//}

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

//func (l *Data) GetName() string {
//  return l.Name
//}
