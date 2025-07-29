package main

import (
	"log"
	"time"

	"github.com/chromedp/chromedp"
)

func runAnalysis() {
	log.Println("=== ITANDI BB HTML Structure Analysis ===")
	
	// Create scraper instance with visible browser for analysis
	scraper, err := NewITANDIScraper(false)
	if err != nil {
		log.Fatal("Failed to create scraper:", err)
	}
	defer scraper.Close()

	// Step 1: Navigate to login page
	log.Println("\n=== Step 1: Analyzing Login Page ===")
	if err := scraper.NavigateToLogin(); err != nil {
		log.Fatal("Failed to navigate:", err)
	}

	time.Sleep(3 * time.Second) // Wait for page to load

	url, _ := scraper.GetPageURL()
	log.Printf("Current URL: %s\n", url)

	// Analyze login page structure
	// AnalyzePageStructure functionality has been removed
	// (DOM saving is no longer needed)

	// Take screenshot
	scraper.TakeScreenshot("analysis_login_page.png")

	// Step 2: Try login and analyze next page
	log.Println("\n=== Step 2: Analyzing Post-Login Page ===")
	if err := scraper.Login(); err != nil {
		log.Printf("Login failed: %v\n", err)
		log.Println("Continuing with analysis of current page...")
	} else {
		log.Println("Login successful, analyzing logged-in page...")
		time.Sleep(3 * time.Second)
	}

	url, _ = scraper.GetPageURL()
	log.Printf("Current URL: %s\n", url)

	// Analyze post-login page
	// AnalyzePageStructure functionality has been removed
	// (DOM saving is no longer needed)

	// Take screenshot
	scraper.TakeScreenshot("analysis_post_login.png")

	// Step 3: Look for search elements
	log.Println("\n=== Step 3: Searching for Search Elements ===")
	
	// Try to find search-related elements with JavaScript
	var searchElements interface{}
	err = chromedp.Run(scraper.ctx,
		chromedp.Evaluate(`
			// Look for potential search elements
			const searchInputs = Array.from(document.querySelectorAll('input')).filter(input => 
				input.type === 'text' || input.type === 'search' ||
				(input.placeholder && (input.placeholder.includes('検索') || input.placeholder.includes('物件') || input.placeholder.includes('search'))) ||
				(input.name && (input.name.includes('search') || input.name.includes('query')))
			);
			
			const searchButtons = Array.from(document.querySelectorAll('button')).filter(btn =>
				btn.textContent.includes('検索') || btn.textContent.includes('Search') ||
				btn.textContent.includes('search') || btn.className.includes('search')
			);
			
			{
				searchInputs: searchInputs.map(el => ({
					tag: el.tagName,
					type: el.type,
					name: el.name,
					id: el.id,
					placeholder: el.placeholder,
					className: el.className
				})),
				searchButtons: searchButtons.map(el => ({
					tag: el.tagName,
					text: el.textContent,
					className: el.className,
					id: el.id
				}))
			}
		`, &searchElements),
	)
	
	if err == nil {
		log.Printf("Search analysis completed\n")
	}

	log.Println("\n=== Analysis Complete ===")
	log.Println("Files generated:")
	log.Println("- page_structure.html: Full HTML structure")
	log.Println("- analysis_login_page.png: Login page screenshot")
	log.Println("- analysis_post_login.png: Post-login screenshot")
	
	// Keep browser open for manual inspection
	log.Println("\nKeeping browser open for 30 seconds for manual inspection...")
	time.Sleep(30 * time.Second)
}