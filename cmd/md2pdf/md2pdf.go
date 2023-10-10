package main

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/jhunters/mdtopdf"
)

var input = flag.String("i", "", "Input filename or HTTP(s) URL; default is os.Stdin")
var output = flag.String("o", "", "Output PDF filename; required")
var pathToSyntaxFiles = flag.String("s", "", "Path to github.com/jessp01/gohighlight/syntax_files")
var title = flag.String("title", "", "Presentation title")
var author = flag.String("author", "", "Author; used if -footer is passed")
var unicodeSupport = flag.String("unicode-encoding", "", "e.g 'cp1251'")
var fontFile = flag.String("font-file", "", "path to font file to use")
var fontName = flag.String("font-name", "Arial", "Font name ID; e.g 'Helvetica-1251'")
var themeArg = flag.String("theme", "light", "[light|dark]")
var hrAsNewPage = flag.Bool("new-page-on-hr", false, "Interpret HR as a new page; useful for presentations")
var printFooter = flag.Bool("with-footer", false, "Print doc footer (author  title  page number)")
var pageSize = flag.String("page-size", "A4", "[A3 | A4 | A5]")
var orientation = flag.String("orientation", "portrait", "[portrait | landscape]")
var logFile = flag.String("log-file", "md2pdf_trace.log", "Path to log file")
var help = flag.Bool("help", false, "Show usage message")
var version = "dev"
var _, fileName, fileLine, ok = runtime.Caller(0)

var opts []mdtopdf.RenderOption

func processRemoteInputFile(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, errors.New("Received non 200 response code: " + fmt.Sprintf("HTTP %d", resp.StatusCode))
	}
	content, rerr := ioutil.ReadAll(resp.Body)
	return content, rerr
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	flag.Parse()

	if *help {
		usage("Help Message")
	}

	if *output == "" {
		usage("Output PDF filename is required")
	}

	if *hrAsNewPage {
		opts = append(opts, mdtopdf.IsHorizontalRuleNewPage(true))
	}

	if *unicodeSupport != "" {
		opts = append(opts, mdtopdf.WithUnicodeTranslator(*unicodeSupport))
	}

	// if *pathToSyntaxFiles != "" {
	// 	opts = append(opts, mdtopdf.SetSyntaxHighlightBaseDir(*pathToSyntaxFiles))
	// } else {
	// 	opts = append(opts, mdtopdf.SetSyntaxHighlightBaseDir("../../highlight/syntax_files"))
	// }

	// get text for PDF
	var content []byte
	var err error
	var inputBaseURL string
	if *input == "" {
		content, err = ioutil.ReadAll(os.Stdin)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		httpRegex := regexp.MustCompile("^http(s)?://")
		if httpRegex.Match([]byte(*input)) {
			content, err = processRemoteInputFile(*input)
			if err != nil {
				log.Fatal(err)
			}
			// get the base URL so we can adjust relative links and images
			inputBaseURL = strings.Replace(filepath.Dir(*input), ":/", "://", 1)
		} else {
			content, err = ioutil.ReadFile(*input)
			if err != nil {
				log.Fatal(err)
			}
		}
	}

	theme := mdtopdf.LIGHT
	textColor := mdtopdf.Colorlookup("black")
	fillColor := mdtopdf.Colorlookup("white")
	backgroundColor := "white"
	if *themeArg == "dark" {
		theme = mdtopdf.DARK
		backgroundColor = "black"
		textColor = mdtopdf.Colorlookup("darkgray")
		fillColor = mdtopdf.Colorlookup("black")
	}

	pf := mdtopdf.NewPdfRenderer(*orientation, *pageSize, *output, *logFile, *fontName, opts, theme)
	if inputBaseURL != "" {
		pf.InputBaseURL = inputBaseURL
	}
	pf.Pdf.SetSubject(*title, true)
	pf.Pdf.SetTitle(*title, true)
	pf.BackgroundColor = mdtopdf.Colorlookup(backgroundColor)

	if *fontName != "" {
		pf.Pdf.AddUTF8Font(*fontName, "", "")
		pf.Pdf.AddUTF8Font(*fontName, "b", "")
		pf.Pdf.AddUTF8Font(*fontName, "i", "")
		pf.Pdf.AddUTF8Font(*fontName, "bi", "")
		// pf.Pdf.SetFont(*fontName, "", 12)
		pf.Normal = mdtopdf.Styler{Font: *fontName, Style: "",
			Size: 12, Spacing: 2,
			FillColor: fillColor,
			TextColor: textColor}
	}

	if *printFooter {
		pf.Pdf.SetFooterFunc(func() {
			color := mdtopdf.Colorlookup(backgroundColor)
			pf.Pdf.SetFillColor(color.Red, color.Green, color.Blue)
			// Position at 1.5 cm from bottom
			pf.Pdf.SetY(-15)
			// Arial italic 8
			pf.Pdf.SetFont("Arial", "I", 8)
			// Text color in gray
			pf.Pdf.SetTextColor(128, 128, 128)
			w, h, _ := pf.Pdf.PageSize(pf.Pdf.PageNo())
			// fmt.Printf("Width: %f, height: %f, unit: %s\n", w, h, u)
			pf.Pdf.SetX(4)
			pf.Pdf.CellFormat(0, 10, fmt.Sprintf("%s", *author), "", 0, "", true, 0, "")
			middle := w / 2
			if *orientation == "landscape" {
				middle = h / 2
			}
			pf.Pdf.SetX(middle - float64(len(*title)))
			pf.Pdf.CellFormat(0, 10, fmt.Sprintf("%s", *title), "", 0, "", true, 0, "")
			pf.Pdf.SetX(-40)
			pf.Pdf.CellFormat(0, 10, fmt.Sprintf("Page %d", pf.Pdf.PageNo()), "", 0, "", true, 0, "")
		})
	}

	err = pf.Process(content)
	if err != nil {
		fmt.Printf("error: %v\n", err)
	}
}

func usage(msg string) {
	fmt.Println(msg + "\n")
	fmt.Printf("Usage: %s (%s) [options]\n", filepath.Base(fileName), version)
	flag.PrintDefaults()
	os.Exit(0)
}
