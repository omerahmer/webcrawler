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

var seen = map[string]bool{}

func filterForAnchor(baseUrl string, node *html.Node) {
	for _, a := range node.Attr {
		if a.Key == "href" {
			normalized, err := normalizeLink(baseUrl, a.Val)
			if err != nil {
				fmt.Println("skipping url:", err)
				return
			}
			if checkIfSeen(normalized) {
				fmt.Println(normalized)
			}
			break
		}
	}
}

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

	resolved := base.ResolveReference(u)
	resolved.Fragment = ""

	return resolved.String(), nil
}

func main() {
	baseUrl := "https://catalogue.uci.edu"
	resp, err := http.Get(baseUrl)
	if err != nil {
		panic("something went wrong")
	}
	defer resp.Body.Close()

	doc, err := html.Parse(resp.Body)

	if err != nil {
		log.Fatal(err)
	}

	for node := range doc.Descendants() {
		if node.Type == html.ElementNode && node.DataAtom == atom.A {
			filterForAnchor(baseUrl, node)
		}
	}
}
