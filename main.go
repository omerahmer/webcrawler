package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

type CrawlItem struct {
	url   string
	depth int
}

var seen = map[string]bool{}

const maxDepth = 2

func checkIfSeen(n string) bool {
	if seen[n] == false {
		seen[n] = true
		return true
	}
	return false
}

func normalizeLink(baseUrl string, href string) (string, error) {
	if href == "" {
		return "", fmt.Errorf("empty href")
	}

	href = strings.TrimSpace(href)

	if href[0] == '#' || strings.HasPrefix(href, "mailto:") || strings.HasPrefix(href, "tel:") {
		return "", fmt.Errorf("invalid fragment or scheme")
	}

	base, err := url.Parse(baseUrl)
	if err != nil {
		return "", fmt.Errorf("invalid base url: %w", err)
	}

	u, err := url.Parse(href)
	if err != nil {
		return "", fmt.Errorf("invalid href: %w", err)
	}
	if u.Scheme != "" && u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("unsupported scheme")
	}

	resolved := base.ResolveReference(u)
	resolved.Fragment = ""

	return resolved.String(), nil
}

func crawlPage(itemUrl string) []string {
	pages := []string{}
	resp, err := http.Get(itemUrl)
	if err != nil {
		return pages
	}
	defer resp.Body.Close()

	doc, err := html.Parse(resp.Body)

	if err != nil {
		log.Fatal(err)
	}

	for node := range doc.Descendants() {
		if node.Type == html.ElementNode && node.DataAtom == atom.A {
			for _, a := range node.Attr {
				if a.Key == "href" {
					normalized, err := normalizeLink(itemUrl, a.Val)
					if err != nil {
						fmt.Printf("skipping href: %q with error %v\n", a.Val, err)
						break
					}
					pages = append(pages, normalized)
					break
				}
			}
		}
	}
	return pages
}

func main() {
	baseUrl := "https://catalogue.uci.edu"

	queue := []CrawlItem{
		{url: baseUrl, depth: 0},
	}

	seen[baseUrl] = true

	seed, err := url.Parse(baseUrl)
	if err != nil {
		log.Fatal(err)
	}

	allowedHost := seed.Host

	for len(queue) > 0 {
		item := queue[0]
		queue = queue[1:]

		if item.depth >= maxDepth {
			continue
		}

		links := crawlPage(item.url)
		for _, link := range links {
			u, err := url.Parse(link)
			if err != nil {
				log.Fatal(err)
			}

			if u.Host != allowedHost {
				continue
			}

			if !seen[link] {
				seen[link] = true
				queue = append(queue, CrawlItem{
					url:   link,
					depth: item.depth + 1,
				})
			}
		}
	}
}
