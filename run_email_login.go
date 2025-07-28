package main

import (
	"log"
	"time"
)

func runEmailLogin() {
	log.Println("=== ITANDI BB Email/Password Login ===")

	// Create email login scraper
	scraper, err := NewEmailLoginScraper(false) // Use visible browser
	if err != nil {
		log.Fatal("Failed to create email scraper:", err)
	}
	defer scraper.Close()

	// Step 1: Find email/password login form
	log.Println("\n=== Step 1: Searching for Email/Password Login Form ===")
	
	if err := scraper.FindEmailLoginForm(); err != nil {
		log.Printf("Could not find email/password login form: %v\n", err)
		log.Println("ITANDI BB may not support traditional email/password login")
		
		// Take screenshot of current state
		scraper.TakeScreenshot("email_login_search_failed.png")
		
		// Show current URL
		url, _ := scraper.GetCurrentURL()
		log.Printf("Current URL: %s\n", url)
		
		log.Println("\nNote: ITANDI BB appears to use phone verification instead of email/password login")
		log.Println("Please use the -updated flag for the phone verification system")
		return
	}

	// Take screenshot of found login form
	scraper.TakeScreenshot("email_login_form_found.png")
	url, _ := scraper.GetCurrentURL()
	log.Printf("Login form found at URL: %s\n", url)

	// Step 2: Perform login
	log.Println("\n=== Step 2: Performing Email/Password Login ===")
	
	email := loginEmail    // From constants
	password := loginPassword // From constants
	
	if err := scraper.PerformEmailLogin(email, password); err != nil {
		log.Printf("Login failed: %v\n", err)
		scraper.TakeScreenshot("email_login_failed.png")
		return
	}

	// Step 3: Verify login success
	log.Println("\n=== Step 3: Verifying Login Success ===")
	
	time.Sleep(3 * time.Second)
	
	// Take screenshot after login
	scraper.TakeScreenshot("email_login_success.png")
	
	url, _ = scraper.GetCurrentURL()
	log.Printf("After login URL: %s\n", url)
	
	// Check if we're on a different page (indicating successful login)
	if url != "https://itandi-accounts.com/" && url != "https://itandi-accounts.com/login" {
		log.Println("✅ Login appears successful - redirected to new page")
		
		// Step 4: Try to search for property
		log.Println("\n=== Step 4: Attempting Property Search ===")
		
		// Look for search functionality
		time.Sleep(2 * time.Second)
		
		// Take final screenshot
		scraper.TakeScreenshot("email_login_dashboard.png")
		
		log.Println("Successfully logged in with email/password!")
		log.Println("You can now implement property search functionality for this interface")
		
	} else {
		log.Println("❌ Login may have failed - still on login page")
	}
	
	log.Println("\n=== Email Login Process Complete ===")
	log.Println("Generated screenshots:")
	log.Println("- email_login_form_found.png")
	log.Println("- email_login_success.png") 
	log.Println("- email_login_dashboard.png")
	
	// Keep browser open for inspection
	log.Println("\nKeeping browser open for 30 seconds for manual inspection...")
	time.Sleep(30 * time.Second)
}