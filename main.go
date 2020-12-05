package main

import (
	"bytes"
	"flag"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"time"

	"github.com/microcosm-cc/bluemonday"
	"github.com/russross/blackfriday/v2"
)

const (
	defaultTemplate = `<!DOCTYPE html
<html>
  <head>
    <meta http-equiv="content-type" content="text/html; charset=utf-8">
    <title>{{ .Title }}</title>
  </head>
  <body>
{{ .Body }}
  </body>
</html>
`
)

type content struct {
	Title string
	Body  template.HTML
}

func main() {
	fileName := flag.String("f", "", "Markdown file to preview")
	skipPreview := flag.Bool("s", false, "Skip auto-preview")
	tFname := flag.String("t", "", "Alternate template name")
	browser := flag.String("b", "firefox", "Browser to preview in")
	flag.Parse()

	if *fileName == "" {
		flag.Usage()
		os.Exit(1)
	}

	if err := run(*fileName, *tFname, os.Stdout, *skipPreview, *browser); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(fileName string, tFname string, out io.Writer, skipPreview bool, browser string) error {
	input, err := ioutil.ReadFile(fileName)
	if err != nil {
		return err
	}

	htmlData, err := parseContent(input, tFname, fileName)
	if err != nil {
		return err
	}

	temp, err := ioutil.TempFile("", "mdp.*.html")
	if err != nil {
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}

	outName := temp.Name()

	fmt.Fprintln(out, outName)

	if err := saveHTML(outName, htmlData); err != nil {
		return err
	}

	if skipPreview {
		return nil
	}

	defer os.Remove(outName)

	return preview(outName, browser)
}

func parseContent(input []byte, tFname string, fileName string) ([]byte, error) {
	output := blackfriday.Run(input)
	body := bluemonday.UGCPolicy().SanitizeBytes(output)

	t, err := template.New("mdp").Parse(defaultTemplate)
	if err != nil {
		return nil, err
	}

	if tFname != "" {
		t, err = template.ParseFiles(tFname)
		if err != nil {
			return nil, err
		}
	}

	c := content{
		Title: fmt.Sprintf("Markdown Preview | %s", fileName),
		Body:  template.HTML(body),
	}

	var buffer bytes.Buffer

	if err := t.Execute(&buffer, c); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func saveHTML(outFname string, data []byte) error {
	return ioutil.WriteFile(outFname, data, 0644)
}

func preview(fname string, browser string) error {
	browserPath, err := exec.LookPath(browser)
	if err != nil {
		return err
	}

	if err := exec.Command(browserPath, fname).Start(); err != nil {
		return err
	}

	// Give the browser some time to open the file before deleting it
	time.Sleep(2 * time.Second)
	return nil
}
