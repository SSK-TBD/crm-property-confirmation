package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/chromedp/chromedp"
)

func findLoginPage() {
	log.Println("=== Searching for Email/Password Login Page ===")
	
	// Create scraper instance with visible browser
	scraper, err := NewITANDIScraper(false)
	if err != nil {
		log.Fatal("Failed to create scraper:", err)
	}
	defer scraper.Close()

	// Try different potential login URLs
	loginURLs := []string{
		"https://itandi-accounts.com/",
		"https://itandi-accounts.com/login",
		"https://itandi-accounts.com/sign_in",
		"https://bukkakun.com/login",
		"https://bukkakun.com/sign_in",
		"https://bukkakun.com/users/sign_in",
		"https://itandi.co.jp/login",
		"https://accounts.itandi.co.jp/",
	}

	for i, url := range loginURLs {
		log.Printf("\n=== Testing URL %d: %s ===\n", i+1, url)
		
		err := chromedp.Run(scraper.ctx,
			chromedp.Navigate(url),
			chromedp.WaitReady("body"),
		)
		
		if err != nil {
			log.Printf("Failed to navigate to %s: %v\n", url, err)
			continue
		}
		
		time.Sleep(3 * time.Second)
		
		// Check for email and password inputs
		var hasEmailInput, hasPasswordInput bool
		
		chromedp.Run(scraper.ctx,
			chromedp.EvaluateAsDevTools(`
				document.querySelector('input[type="email"], input[name="email"], input[id*="email"]') !== null
			`, &hasEmailInput),
		)
		
		chromedp.Run(scraper.ctx,
			chromedp.EvaluateAsDevTools(`
				document.querySelector('input[type="password"], input[name="password"], input[id*="password"]') !== null
			`, &hasPasswordInput),
		)
		
		currentURL, _ := scraper.GetPageURL()
		log.Printf("Current URL: %s\n", currentURL)
		log.Printf("Has Email Input: %v\n", hasEmailInput)
		log.Printf("Has Password Input: %v\n", hasPasswordInput)
		
		if hasEmailInput && hasPasswordInput {
			log.Printf("✅ Found login form at: %s\n", url)
			
			// Take screenshot
			filename := "login_found_" + fmt.Sprintf("%d", i+1) + ".png"
			scraper.TakeScreenshot(filename)
			
			// Get page HTML
			var html string
			chromedp.Run(scraper.ctx,
				chromedp.OuterHTML("html", &html, chromedp.ByQuery),
			)
			
			// Save HTML
			htmlFilename := "login_page_" + fmt.Sprintf("%d", i+1) + ".html"
			os.WriteFile(htmlFilename, []byte(html), 0644)
			
			log.Printf("Screenshots and HTML saved for %s\n", url)
		} else {
			log.Printf("❌ No standard login form found at: %s\n", url)
		}
		
		time.Sleep(2 * time.Second)
	}
	
	log.Println("\n=== Search Complete ===")
	log.Println("Check the generated screenshots and HTML files for login forms")
	
	// Keep browser open for manual inspection
	log.Println("\nKeeping browser open for 30 seconds for manual inspection...")
	time.Sleep(30 * time.Second)
}