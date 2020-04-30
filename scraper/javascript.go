package scraper

import (
	"strings"
	"text/template"
)

const (
	// This code selects the last pin and brings it to the visible part of the
	// browser window. This causes the page to load further pin's.
	tplJsScrollIntoView string = `
var allCurrentPreviewPictures = document.querySelectorAll('{{.SelectorPreviewPins}}');
var lastPicture = allCurrentPreviewPictures[allCurrentPreviewPictures.length - 1];
lastPicture.scrollIntoView(true);
[].map.call(allCurrentPreviewPictures,img => (img.srcset));
`
)

// renderJsScrollIntoView returns a rendered JavaScript code snippet. The code
// is used to select the last picture and scroll it into view. As already mentioned
// above this causes the page to load further pins's.
func renderJsScrollIntoView(selectorPreviewPins string) func() string {
	var jsScrollIntoView string

	return func() string {
		if jsScrollIntoView == "" {
			var sb strings.Builder

			data := struct {
				SelectorPreviewPins string
			}{
				SelectorPreviewPins: selectorPreviewPins,
			}

			tpl := template.Must(template.New("jsScrollIntoView").Parse(tplJsScrollIntoView))

			err := tpl.Execute(&sb, data)
			if err != nil {
				// If we can't parse the JS template we can panic as the whole
				// program becomes useless anyways in that case.
				panic(err)
			}

			renderedTpl := sb.String()
			jsScrollIntoView = renderedTpl

		}

		return jsScrollIntoView
	}
}
