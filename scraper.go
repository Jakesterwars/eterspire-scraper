package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"eterspire-scraper/helpers"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly"
)

type Section struct {
	name   string
	values []string
}

type PatchNotes struct {
	date, bigImage string
	sections       []Section
}

func main() {
	// No arguments provided, exit with a message
	if len(os.Args) <= 1 {
		fmt.Println("Please provide a valid URL to scrape")
		return
	}

	args := os.Args
	pageURL := args[1]
	collector := colly.NewCollector()

	patch := PatchNotes{}
	section := Section{}

	collector.OnRequest(func(r *colly.Request) {
		fmt.Println("Visiting", r.URL)
	})

	collector.OnResponse(func(r *colly.Response) {
		fmt.Println("Got a response from", r.Request.URL)
	})

	collector.OnError(func(r *colly.Response, e error) {
		fmt.Println("Error occurred:", e)
	})

	collector.OnHTML(".eterspire-border", func(e *colly.HTMLElement) {
		// Check if actual body of patch notes
		if e.ChildText(".section-title-news") == "" {
			return
		}

		// Replace the default opening and closing [ ] around the date for future use
		dateReplacer := strings.NewReplacer("[", "", "]", "")

		// Setup base patch note information
		patch.date = dateReplacer.Replace(e.ChildText(".section-date"))
		patch.bigImage = filepath.Base(e.ChildAttr("img", "src"))
		imageURL := helpers.ResolveImageURL(e.Request.URL.String(), e.ChildAttr("img", "src"))
		if imageURL != "" {
			helpers.DownloadImage(imageURL, patch.date+"-downloads")
		} else {
			fmt.Println("No valid image source found for header image")
		}

		e.DOM.Find("h4").Each(func(_ int, h4Elem *goquery.Selection) {
			section = Section{}
			section.name = strings.TrimSpace(h4Elem.Text())

			content := h4Elem.NextUntil("h4")

			content.Filter("center, ul, p, div").Each(func(_ int, contentElems *goquery.Selection) {
				tagName := strings.ToLower(contentElems.Nodes[0].Data)

				switch tagName {
				case "center":
					if h4Elem.Text() == "Fanart" {
						section.values = append(section.values, "<gallery mode=\"nolines\" widths=\"200px\" heights=\"200px\">")
						contentElems.Find("figure").Each(func(_ int, figureElem *goquery.Selection) {
							var figCaption = figureElem.Find("figcaption").Text()
							var imageBase = filepath.Base(figureElem.Find("img").AttrOr("src", ""))
							var trimmedImageName = strings.TrimSuffix(imageBase, filepath.Ext(imageBase))
							imageURL := helpers.ResolveImageURL(e.Request.URL.String(), figureElem.Find("img").AttrOr("src", ""))
							if imageURL != "" {
								helpers.DownloadImage(imageURL, patch.date+"-downloads")
							} else {
								fmt.Println("No valid image source found for header image")
							}

							section.values = append(section.values, "File:"+trimmedImageName+" "+patch.date+".png | "+figCaption)
						})
						section.values = append(section.values, "</gallery>")
					} else {
						contentElems.Find("img").Each(func(_ int, imgElem *goquery.Selection) {
							if filepath.Base(imgElem.AttrOr("src", "")) == "creators_spotlight_separator.png" {
								section.values = append(section.values, "{{Creator Spotlight Banner}}\n")
							} else {
								imageURL := helpers.ResolveImageURL(e.Request.URL.String(), imgElem.AttrOr("src", ""))
								if imageURL != "" {
									helpers.DownloadImage(imageURL, patch.date+"-downloads")
								} else {
									fmt.Println("No valid image source found for header image")
								}
								section.values = append(section.values, "{{NewsImage|"+filepath.Base(imgElem.AttrOr("src", ""))+"}}\n")
							}
						})
					}

				case "ul":
					contentElems.Each(func(_ int, ulElem *goquery.Selection) {
						ulElem.Find("li").Each(func(index int, liElem *goquery.Selection) {
							line := helpers.FormatSelectionInline(liElem)
							if line != "" {
								var lastElement = ""
								if index == ulElem.Find("li").Length()-1 {
									lastElement = "\n"
								}
								section.values = append(section.values, "* "+line+lastElement)
							}
						})
					})

				case "p":
					line := helpers.FormatSelectionInline(contentElems)
					if line != "" {
						section.values = append(section.values, line+"\n")
					}

				case "div":
					contentElems.Find("iframe").Each(func(_ int, iframeElem *goquery.Selection) {
						section.values = append(section.values, "{{Youtube|link="+filepath.Base(iframeElem.AttrOr("src", ""))+"|align=center}}\n")
					})
				}
			})

			patch.sections = append(patch.sections, section)
		})
	})

	collector.Visit(pageURL)

	file, err := os.Create(patch.date + "-patchnotes.txt")
	if err != nil {
		fmt.Println("Error creating file:", err)
		return
	}
	defer file.Close()

	fmt.Fprintln(file, "{{NewsPost|date="+patch.date+"|archive=}}")
	fmt.Fprintln(file, "{{NewsDate|date="+patch.date+"}}")
	fmt.Fprintln(file, "\n{{NewsImage|"+patch.bigImage+"}}\n")

	for _, section := range patch.sections {
		fmt.Fprintln(file, "== "+section.name+" ==")
		for _, note := range section.values {
			fmt.Fprintln(file, note)
		}
	}
}
