# eterspire-scraper

Requirements to run:

1. Go installed - https://go.dev/doc/install

Example run:

`go run scraper.go https://www.eterspire.com/news/?date=2026-06-16`

This will create two things:

1. A downloads folder named DATE-downloads that will house all the images it rips from the website during processing. These will be uploaded to the wiki manually.
2. A DATE-patchnotes.txt file will be generated you will use to create the Patch notes page.

Upload the url that was provided to https://web.archive.org/ to fill in the archive parameter of NewsPost.

Upload the images, upload the text generated, and be golden with a simple patch notes day!
