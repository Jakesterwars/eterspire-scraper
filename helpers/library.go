package helpers

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html"
)

// Get the value of an attribute from an HTML node, case-insensitive
func attrValue(node *html.Node, key string) string {
	for _, attr := range node.Attr {
		if strings.EqualFold(attr.Key, key) {
			return attr.Val
		}
	}

	return ""
}

// Normalize the inline text by removing extra whitespace and fixing punctuation spacing
func normalizeInline(s string) string {
	s = html.UnescapeString(s)
	s = strings.Join(strings.Fields(s), " ")

	punct := strings.NewReplacer(
		" ,", ",",
		" .", ".",
		" !", "!",
		" ?", "?",
		" ;", ";",
		" :", ":",
		" )", ")",
		"( ", "(",
	)

	return strings.TrimSpace(punct.Replace(s))
}

// Render the inline content of an HTML node to a string, handling specific tags for formatting
func renderInline(node *html.Node) string {
	if node == nil {
		return ""
	}

	switch node.Type {
	case html.TextNode:
		return node.Data
	case html.ElementNode:
		tag := strings.ToLower(node.Data)
		var inner strings.Builder

		for child := node.FirstChild; child != nil; child = child.NextSibling {
			inner.WriteString(renderInline(child))
		}

		content := inner.String()

		switch tag {
		case "a":
			href := attrValue(node, "href")
			label := normalizeInline(content)
			if href == "" {
				return label
			}
			return "[" + href + " '''" + label + "''']"
		case "strong", "b":
			return "'''" + content + "'''"
		case "em", "i":
			return "''" + content + "''"
		case "br":
			return "\n"
		default:
			return content
		}
	default:
		return ""
	}
}

// Format the inline content of a goquery selection to a string
func FormatSelectionInline(sel *goquery.Selection) string {
	if sel == nil || len(sel.Nodes) == 0 {
		return ""
	}

	var out strings.Builder
	for _, node := range sel.Nodes {
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			out.WriteString(renderInline(child))
		}
	}

	return normalizeInline(out.String())
}

func DownloadImage(imageURL, dir string) {
	fmt.Println("Downloading:", imageURL)

	if err := os.MkdirAll(dir, 0o755); err != nil {
		fmt.Printf("Failed to create directory %s: %v\n", dir, err)
		return
	}

	// Make the HTTP GET request
	resp, err := http.Get(imageURL)
	if err != nil {
		fmt.Printf("Failed to download %s: %v\n", imageURL, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		fmt.Printf("Failed to download %s, status code: %d\n", imageURL, resp.StatusCode)
		return
	}

	// Create the local file
	parsedURL, err := url.Parse(imageURL)
	if err != nil {
		fmt.Printf("Failed to parse image URL %s: %v\n", imageURL, err)
		return
	}

	base := filepath.Base(parsedURL.Path)
	if base == "." || base == "/" || base == "" {
		base = "image"
	}

	fileName, err := url.PathUnescape(base)
	if err != nil {
		fileName = base
	}
	filePath := filepath.Join(dir, fileName)

	file, err := os.Create(filePath)
	if err != nil {
		fmt.Printf("Failed to create file %s: %v\n", filePath, err)
		return
	}
	defer file.Close()

	// Write the downloaded content to the file
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		fmt.Printf("Failed to save image to %s: %v\n", filePath, err)
		return
	}

	fmt.Printf("Saved image to %s\n", filePath)
}

func ResolveImageURL(pageURL, rawSrc string) string {
	rawSrc = strings.TrimSpace(rawSrc)
	if rawSrc == "" {
		return ""
	}

	parsedSrc, err := url.Parse(rawSrc)
	if err != nil {
		return ""
	}
	if parsedSrc.IsAbs() {
		return parsedSrc.String()
	}

	base, err := url.Parse(pageURL)
	if err != nil {
		return ""
	}

	return base.ResolveReference(parsedSrc).String()
}
