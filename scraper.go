package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

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

		const invalidImageSource = "No valid image source found for header image"

		// Replace the default opening and closing [ ] around the date for future use
		dateReplacer := strings.NewReplacer("[", "", "]", "")

		// Setup base patch note information
		patch.date = dateReplacer.Replace(e.ChildText(".section-date"))
		downloadsDir := patch.date + "-downloads"
		dateSuffix := " " + patch.date
		patch.bigImage = filepath.Base(e.ChildAttr("img", "src"))
		imageURL := helpers.ResolveImageURL(e.Request.URL.String(), e.ChildAttr("img", "src"))
		if imageURL != "" {
			helpers.DownloadImage(imageURL, downloadsDir, "")
		} else {
			fmt.Println(invalidImageSource)
		}

		downloadSectionImage := func(rawSrc string) {
			imageURL := helpers.ResolveImageURL(e.Request.URL.String(), rawSrc)
			if imageURL != "" {
				helpers.DownloadImage(imageURL, downloadsDir, dateSuffix)
			} else {
				fmt.Println(invalidImageSource)
			}
		}

		buildSectionImageName := func(rawSrc string) string {
			return helpers.BuildAppendedFileName(filepath.Base(rawSrc), dateSuffix)
		}

		isSpotlightSeparator := func(rawSrc string) bool {
			return filepath.Base(rawSrc) == "creators_spotlight_separator.png"
		}

		e.DOM.Find("h4").Each(func(_ int, h4Elem *goquery.Selection) {
			section = Section{}
			section.name = strings.TrimSpace(h4Elem.Text())

			content := h4Elem.NextUntil("h4")

			content.Filter("center, ul, p, div, h3").Each(func(_ int, contentElems *goquery.Selection) {
				tagName := strings.ToLower(contentElems.Nodes[0].Data)

				switch tagName {
				case "h3":
					section.values = append(section.values, "<center>'''"+strings.TrimSpace(contentElems.Text())+"'''</center>\n")

				case "center":
					if h4Elem.Text() == "Fanart" {
						section.values = append(section.values, "<gallery mode=\"nolines\" widths=\"200px\" heights=\"200px\">")
						contentElems.Find("figure").Each(func(_ int, figureElem *goquery.Selection) {
							figCaption := helpers.FormatSelectionInline(figureElem.Find("figcaption"), patch.date)
							rawSrc := figureElem.Find("img").AttrOr("src", "")
							imageBase := filepath.Base(rawSrc)
							trimmedImageName := strings.TrimSuffix(imageBase, filepath.Ext(imageBase))
							downloadSectionImage(rawSrc)

							section.values = append(section.values, "File:"+trimmedImageName+" "+patch.date+".png | "+figCaption)
						})
						section.values = append(section.values, "</gallery>")
					} else {
						processCenterImage := func(rawSrc string, figCaption string) {
							if rawSrc == "" {
								return
							}

							if isSpotlightSeparator(rawSrc) {
								section.values = append(section.values, "{{Creator Spotlight Banner}}\n")
								return
							}

							downloadSectionImage(rawSrc)
							imageName := buildSectionImageName(rawSrc)
							if figCaption != "" {
								section.values = append(section.values, "{{NewsImage|"+imageName+"}}\n<center>"+figCaption+"</center>\n")
								return
							}

							section.values = append(section.values, "{{NewsImage|"+imageName+"}}\n")
						}

						processList := func(ulElem *goquery.Selection) {
							ulElem.Find("li").Each(func(index int, liElem *goquery.Selection) {
								line := helpers.FormatSelectionInline(liElem, patch.date)
								if line == "" {
									return
								}

								lastElement := ""
								if index == ulElem.Find("li").Length()-1 {
									lastElement = "\n"
								}

								section.values = append(section.values, "* "+line+lastElement)
							})
						}

						processParagraph := func(pElem *goquery.Selection) {
							line := helpers.FormatSelectionInline(pElem, patch.date)
							if line != "" {
								section.values = append(section.values, line+"\n")
							}
						}

						processDiv := func(divElem *goquery.Selection) {
							divElem.Find("iframe").Each(func(_ int, iframeElem *goquery.Selection) {
								section.values = append(section.values, "{{Youtube|link="+filepath.Base(iframeElem.AttrOr("src", ""))+"|align=center}}\n")
							})
						}

						contentElems.Children().Each(func(_ int, child *goquery.Selection) {
							tag := goquery.NodeName(child)

							switch tag {
							case "figure":
								rawSrc := child.Find("img").First().AttrOr("src", "")
								figCaption := helpers.FormatSelectionInline(child.Find("figcaption"), patch.date)
								processCenterImage(rawSrc, figCaption)

							case "img":
								rawSrc := child.AttrOr("src", "")
								processCenterImage(rawSrc, "")

							case "p":
								processParagraph(child)

							case "ul":
								processList(child)

							case "div":
								processDiv(child)

							case "br":
								// Ignore soft breaks so they do not produce empty output entries.
							}
						})
					}

				case "ul":
					contentElems.Each(func(_ int, ulElem *goquery.Selection) {
						ulElem.Find("li").Each(func(index int, liElem *goquery.Selection) {
							line := helpers.FormatSelectionInline(liElem, patch.date)
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
					line := helpers.FormatSelectionInline(contentElems, patch.date)
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
	fmt.Fprintln(file, "{{NewsDate|"+patch.date+"}}")
	fmt.Fprintln(file, "\n{{NewsImage|"+patch.bigImage+"}}\n")

	for _, section := range patch.sections {
		fmt.Fprintln(file, "{{NewsHeader|"+section.name+"}}")
		for _, note := range section.values {
			fmt.Fprintln(file, note)
		}
	}

	fmt.Fprintln(file, "\n=== Developer Teasers ===\n<gallery>\n</gallery>")

	layout := "2006-01-02"
	parsedDate, err := time.Parse(layout, patch.date)
	if err != nil {
		fmt.Println("Error parsing date:", err)
		return
	}
	year := parsedDate.Year()
	fmt.Fprintln(file, "\n[[Category:"+strconv.Itoa(year)+" news]]")
}
