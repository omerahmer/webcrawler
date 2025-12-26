package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/jimsmart/grobotstxt"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

var sharedClient = &http.Client{
	Timeout: 3 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:        200,
		MaxIdleConnsPerHost: 100,
		IdleConnTimeout:     90 * time.Second,
	},
}

var (
	seen        = make(map[string]bool)
	mu          sync.Mutex
	maxPages    int
	allowedHost string
	robotsCache = make(map[string]string)
	robotsMu    sync.Mutex
)

func normalizeLink(baseUrl string, href string) (string, error) {
	if href == "" {
		return "", fmt.Errorf("empty href")
	}
	href = strings.TrimSpace(href)
	if len(href) > 0 && (href[0] == '#' || strings.HasPrefix(href, "mailto:") || strings.HasPrefix(href, "tel:")) {
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

func crawlPage(itemUrl string) ([]string, error) {
	resp, err := sharedClient.Get(itemUrl)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	pages := []string{}
	tokenizer := html.NewTokenizer(resp.Body)
	for {
		tokenType := tokenizer.Next()
		if tokenType == html.ErrorToken {
			err := tokenizer.Err()
			if err == io.EOF {
				break
			}
			log.Printf("error tokenizing %s: %v", itemUrl, err)
			break
		}
		if tokenType == html.StartTagToken {
			token := tokenizer.Token()
			if token.DataAtom == atom.A {
				for _, a := range token.Attr {
					if a.Key == "href" {
						normalized, err := normalizeLink(itemUrl, a.Val)
						if err == nil {
							pages = append(pages, normalized)
						}
						break
					}
				}
			}
		}
	}
	return pages, nil
}

func crawl(itemUrl string, wg *sync.WaitGroup, sem chan struct{}) {
	defer wg.Done()

	sem <- struct{}{}
	defer func() { <-sem }()

	fmt.Printf("visited: %s\n", itemUrl)

	links, err := crawlPage(itemUrl)
	if err != nil {
		return
	}

	for _, link := range links {
		u, err := url.Parse(link)
		if err != nil || u.Host != allowedHost {
			continue
		}

		scheme := u.Scheme
		host := u.Host
		var robotsTxt string

		robotsMu.Lock()
		cached, ok := robotsCache[host]
		if ok {
			robotsTxt = cached
		} else {
			robotsUrl := scheme + "://" + host + "/robots.txt"
			resp, err := sharedClient.Get(robotsUrl)
			if err != nil {
				log.Print("no robots.txt found, continuing")
			}
			defer resp.Body.Close()

			bodyBytes, err := io.ReadAll(resp.Body)
			if err != nil {
				log.Fatalf("failed to read robots.txt, err: %w", err)
			}
			robotsTxt = string(bodyBytes)
			robotsCache[host] = robotsTxt
		}
		robotsMu.Unlock()

		allowed := grobotstxt.AgentAllowed(robotsTxt, "MyCrawler/1.0", link)

		if !allowed {
			continue
		}

		mu.Lock()
		if seen[link] || len(seen) >= maxPages {
			mu.Unlock()
			continue
		}
		seen[link] = true
		mu.Unlock()

		wg.Add(1)
		go crawl(link, wg, sem)
	}
}

func main() {
	fmt.Println("Enter URL you want to crawl (must start with \"https://\"): ")
	baseUrl := ""
	fmt.Scan(&baseUrl)

	fmt.Println("Enter max number of pages you want to crawl: ")
	fmt.Scan(&maxPages)

	start := time.Now()

	seed, err := url.Parse(baseUrl)
	if err != nil {
		log.Fatal(err)
	}
	allowedHost = seed.Host

	if err != nil {
		log.Fatal("Failed to fetch robots.txt with error: %w", err)
	}

	seen[baseUrl] = true
	var wg sync.WaitGroup
	sem := make(chan struct{}, 200)

	wg.Add(1)
	go crawl(baseUrl, &wg, sem)

	wg.Wait()

	elapsed := time.Since(start)
	log.Printf("Crawling took %s", elapsed)
	log.Printf("Total pages crawled: %d", len(seen))
}
