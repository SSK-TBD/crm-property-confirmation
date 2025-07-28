package main

import (
	"log"
	"time"
	
	"github.com/chromedp/chromedp"
)

func testModalHandling() {
	log.Println("=== Testing Modal Advertisement Handling ===")
	
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
	
	// Navigate to search page directly to test modal handling
	log.Println("Navigating to search page...")
	err = chromedp.Run(scraper.ctx,
		chromedp.Navigate("https://itandibb.com/rent_rooms/list"),
		chromedp.WaitReady("body"),
	)
	if err != nil {
		log.Fatal("Failed to navigate to search page:", err)
	}
	
	time.Sleep(3 * time.Second)
	
	// Test modal closing
	log.Println("Testing modal advertisement closing...")
	err = scraper.closeModalAds()
	if err != nil {
		log.Printf("Modal handling error: %v\n", err)
	}
	
	// Take screenshot
	scraper.TakeScreenshot("test_modal_after_close.png")
	
	log.Println("Keeping browser open for inspection...")
	time.Sleep(30 * time.Second)
}