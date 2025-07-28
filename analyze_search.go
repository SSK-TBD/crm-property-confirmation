package main

import (
	"log"
	"time"

	"github.com/chromedp/chromedp"
)

func analyzeSearchFlow() {
	log.Println("=== Analyzing ITANDI BB Search Flow ===")
	
	// Create scraper instance with visible browser
	scraper, err := NewITANDIScraper(false)
	if err != nil {
		log.Fatal("Failed to create scraper:", err)
	}
	defer scraper.Close()

	// Step 1: Navigate and login
	log.Println("\n=== Step 1: Login ===")
	if err := scraper.NavigateToLogin(); err != nil {
		log.Fatal("Failed to navigate:", err)
	}

	time.Sleep(2 * time.Second)

	if err := scraper.Login(); err != nil {
		log.Fatal("Failed to login:", err)
	}

	time.Sleep(5 * time.Second)
	
	// Step 2: Analyze top page structure
	log.Println("\n=== Step 2: Analyzing Top Page ===")
	url, _ := scraper.GetPageURL()
	log.Printf("Current URL: %s\n", url)
	
	// Take screenshot
	scraper.TakeScreenshot("analyze_top_page.png")
	
	// Look for rental module and list search
	var rentalLinks []interface{}
	err = chromedp.Run(scraper.ctx,
		chromedp.Evaluate(`
			const links = Array.from(document.querySelectorAll('a'));
			links.map(a => ({
				text: a.textContent.trim(),
				href: a.href,
				className: a.className,
				id: a.id,
				hasRental: a.textContent.includes('賃貸'),
				hasSearch: a.textContent.includes('検索'),
				hasList: a.textContent.includes('リスト')
			})).filter(a => a.hasRental || a.hasSearch || a.hasList)
		`, &rentalLinks),
	)
	
	if err == nil && len(rentalLinks) > 0 {
		log.Printf("Found %d rental/search related links\n", len(rentalLinks))
		for i, link := range rentalLinks {
			log.Printf("Link %d: %+v\n", i+1, link)
		}
	}
	
	// Look for modules
	var modules []interface{}
	err = chromedp.Run(scraper.ctx,
		chromedp.Evaluate(`
			const modules = Array.from(document.querySelectorAll('[class*="module"], [class*="賃貸"], div[id*="rental"]'));
			modules.map(m => ({
				className: m.className,
				id: m.id,
				text: m.textContent.substring(0, 100),
				hasLinks: m.querySelectorAll('a').length
			}))
		`, &modules),
	)
	
	if err == nil && len(modules) > 0 {
		log.Printf("Found %d modules\n", len(modules))
		for i, module := range modules {
			log.Printf("Module %d: %+v\n", i+1, module)
		}
	}
	
	// Step 3: Try to click list search
	log.Println("\n=== Step 3: Clicking List Search ===")
	
	// Try the improved search function
	err = scraper.SearchProperty("テスト物件")
	if err != nil {
		log.Printf("Search failed: %v\n", err)
		
		// Take screenshot of current state
		scraper.TakeScreenshot("analyze_search_failed.png")
		
		// Try manual analysis
		log.Println("Performing manual link analysis...")
		
		var allLinks []interface{}
		chromedp.Run(scraper.ctx,
			chromedp.Evaluate(`
				Array.from(document.querySelectorAll('a')).map(a => ({
					text: a.textContent.trim(),
					href: a.href,
					visible: a.offsetParent !== null
				})).filter(a => a.visible && a.text.length > 0)
			`, &allLinks),
		)
		
		log.Printf("Found %d visible links\n", len(allLinks))
		for i, link := range allLinks {
			if i < 20 { // Show first 20 links
				log.Printf("Visible link %d: %+v\n", i+1, link)
			}
		}
	} else {
		log.Println("Search initiated successfully")
		scraper.TakeScreenshot("analyze_search_success.png")
	}
	
	log.Println("\n=== Analysis Complete ===")
	log.Println("Generated screenshots:")
	log.Println("- analyze_top_page.png")
	log.Println("- analyze_search_failed.png or analyze_search_success.png")
	
	// Keep browser open for manual inspection
	log.Println("\nKeeping browser open for 60 seconds for manual inspection...")
	time.Sleep(60 * time.Second)
}