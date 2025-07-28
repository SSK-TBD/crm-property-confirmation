package main

import (
	"log"
	"time"

	"github.com/chromedp/chromedp"
)

func analyzeDetailedSearch() {
	log.Println("=== Detailed Search Result Analysis ===")
	
	// Create scraper instance with visible browser
	scraper, err := NewITANDIScraper(false)
	if err != nil {
		log.Fatal("Failed to create scraper:", err)
	}
	defer scraper.Close()

	// Navigate and login
	if err := scraper.NavigateToLogin(); err != nil {
		log.Fatal("Failed to navigate:", err)
	}

	time.Sleep(2 * time.Second)

	if err := scraper.Login(); err != nil {
		log.Fatal("Failed to login:", err)
	}

	time.Sleep(5 * time.Second)
	
	// Search for a property
	if err := scraper.SearchProperty("クレール立川"); err != nil {
		log.Fatal("Failed to search:", err)
	}

	time.Sleep(3 * time.Second)
	
	// Analyze the search result page structure
	log.Println("\n=== Analyzing Search Result Structure ===")
	
	// Get table structure
	var tableInfo interface{}
	err = chromedp.Run(scraper.ctx,
		chromedp.Evaluate(`
			const tables = document.querySelectorAll('table');
			const tableData = Array.from(tables).map(table => {
				const headers = Array.from(table.querySelectorAll('th')).map(th => th.textContent.trim());
				const rows = Array.from(table.querySelectorAll('tbody tr')).slice(0, 3).map(tr => {
					return Array.from(tr.querySelectorAll('td')).map(td => ({
						text: td.textContent.trim(),
						className: td.className,
						dataLabel: td.getAttribute('data-label')
					}));
				});
				return {
					className: table.className,
					id: table.id,
					headers: headers,
					rowCount: table.querySelectorAll('tbody tr').length,
					sampleRows: rows
				};
			});
			tableData
		`, &tableInfo),
	)
	
	if err == nil {
		log.Printf("Table structure: %+v\n", tableInfo)
	}
	
	// Get property links
	var propertyLinks []interface{}
	err = chromedp.Run(scraper.ctx,
		chromedp.Evaluate(`
			Array.from(document.querySelectorAll('a[href*="/rent_rooms/"]'))
				.filter(a => !a.href.includes('/list'))
				.map(a => ({
					text: a.textContent.trim(),
					href: a.href,
					className: a.className,
					parent: a.parentElement.tagName
				}))
		`, &propertyLinks),
	)
	
	if err == nil && len(propertyLinks) > 0 {
		log.Printf("Found %d property links\n", len(propertyLinks))
		for i, link := range propertyLinks {
			if i < 5 {
				log.Printf("Property link %d: %+v\n", i+1, link)
			}
		}
	}
	
	// Get specific elements by class
	var elements []interface{}
	err = chromedp.Run(scraper.ctx,
		chromedp.Evaluate(`
			const selectors = [
				'[class*="property"]',
				'[class*="building"]',
				'[class*="rent"]',
				'[class*="address"]',
				'[class*="room"]'
			];
			
			const results = [];
			selectors.forEach(selector => {
				const els = document.querySelectorAll(selector);
				Array.from(els).slice(0, 3).forEach(el => {
					results.push({
						selector: selector,
						tagName: el.tagName,
						className: el.className,
						text: el.textContent.trim().substring(0, 50)
					});
				});
			});
			results
		`, &elements),
	)
	
	if err == nil && len(elements) > 0 {
		log.Printf("Found %d elements with property-related classes\n", len(elements))
		for i, element := range elements {
			if i < 10 {
				log.Printf("Element %d: %+v\n", i+1, element)
			}
		}
	}
	
	// Take detailed screenshot
	scraper.TakeScreenshot("detailed_search_results.png")
	
	log.Println("\n=== Analysis Complete ===")
	log.Println("Keeping browser open for manual inspection...")
	time.Sleep(60 * time.Second)
}