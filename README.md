# Go Web Crawler

A concurrent web crawler written in Go that respects `robots.txt` rules, fetches links recursively, and limits concurrent requests.

## Features Implemented So Far

- **Concurrency**
  - Uses goroutines and a semaphore to limit concurrency.
  - WaitGroup ensures all URLs are fully processed before the program exits.

- **URL normalization and deduplication**
  - Normalizes relative and absolute URLs.
  - Ignores invalid schemes, fragments, `mailto:`, and `tel:` links.
  - Tracks visited URLs using a thread-safe map to prevent revisiting.

- **robots.txt compliance**
  - Fetches `robots.txt` for each host.
  - Uses `github.com/jimsmart/grobotstxt` to check if crawling is allowed.
  - Caches `robots.txt` per host to avoid repeated requests.

- **Configurable limits**
  - User can specify the maximum number of pages to crawl.
  - Limits concurrent requests with a semaphore (configurable buffer size).

- **HTML parsing**
  - Extracts `<a href>` links from pages using `golang.org/x/net/html`.
  - Ignores script, style, and non-HTTP links.

- **Polite crawling considerations**
  - Respects host restrictions (does not follow external links).
  - Includes timeout and idle connection settings in the HTTP client.

- **Logging**
  - Prints visited URLs in real time.
  - Reports total pages crawled and total time taken at the end.
