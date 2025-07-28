package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"
)

func runUpdatedScraper() {
	log.Println("=== ITANDI BB Updated Scraper ===")
	
	// Use default values since flags are already parsed in main
	companyName := "クレール" // Default company name
	propertyName := "サンプル物件" // Default property name
	headless := false // Default to visible mode

	// Create scraper instance
	scraper, err := NewITANDIScraperUpdated(headless)
	if err != nil {
		log.Fatal("Failed to create scraper:", err)
	}
	defer scraper.Close()

	// Step 1: Navigate to login/verification page
	log.Println("\n=== Step 1: Navigating to ITANDI BB ===")
	if err := scraper.NavigateToLogin(); err != nil {
		log.Fatal("Failed to navigate:", err)
	}

	time.Sleep(3 * time.Second)
	
	url, _ := scraper.GetCurrentURL()
	log.Printf("Current URL: %s\n", url)

	// Take screenshot
	if err := scraper.TakeScreenshot("updated_step1_initial_page.png"); err != nil {
		log.Println("Warning: Failed to take screenshot:", err)
	}

	// Step 2: Handle phone verification process
	log.Printf("\n=== Step 2: Processing Phone Verification for '%s' ===\n", companyName)
	if err := scraper.ProcessPhoneVerification(companyName); err != nil {
		log.Printf("Phone verification failed: %v\n", err)
		log.Println("This is expected if the company is not found or phone verification is required")
		
		// Take screenshot of the current state
		scraper.TakeScreenshot("updated_step2_verification_state.png")
		
		// Try to get phone number if available
		phoneNumber, err := scraper.GetPhoneNumber()
		if err == nil {
			log.Printf("Phone number for verification: %s\n", phoneNumber)
			log.Println("Manual phone verification would be required here")
		}
		
		// Continue with current page analysis
		log.Println("Continuing with analysis of current page...")
	} else {
		log.Println("Phone verification process initiated successfully")
		
		// Take screenshot
		scraper.TakeScreenshot("updated_step2_after_verification.png")
		
		time.Sleep(5 * time.Second)
	}

	// Step 3: Try to navigate to property search (if logged in)
	log.Printf("\n=== Step 3: Attempting Property Search for '%s' ===\n", propertyName)
	
	url, _ = scraper.GetCurrentURL()
	log.Printf("Current URL: %s\n", url)
	
	if err := scraper.SearchPropertyInUpdatedInterface(propertyName); err != nil {
		log.Printf("Property search failed: %v\n", err)
		log.Println("This might be due to not being fully logged in or interface differences")
	} else {
		log.Println("Property search completed")
	}

	// Take screenshot of search results or current state
	scraper.TakeScreenshot("updated_step3_search_state.png")

	// Step 4: Try to extract any available property information
	log.Println("\n=== Step 4: Extracting Available Information ===")
	details, err := scraper.GetUpdatedPropertyDetails()
	if err != nil {
		log.Printf("Failed to get property details: %v\n", err)
	} else if len(details) > 0 {
		// Print details in JSON format
		jsonData, _ := json.MarshalIndent(details, "", "  ")
		fmt.Printf("\nExtracted Information (JSON):\n%s\n", jsonData)
		
		// Also print in readable format
		fmt.Println("\nExtracted Information:")
		for key, value := range details {
			fmt.Printf("- %s: %s\n", key, value)
		}
	} else {
		log.Println("No property details could be extracted from current page")
	}

	// Take final screenshot
	scraper.TakeScreenshot("updated_step4_final_state.png")

	log.Println("\n=== Updated Scraper Complete ===")
	log.Println("Files generated:")
	log.Println("- updated_step1_initial_page.png")
	log.Println("- updated_step2_verification_state.png") 
	log.Println("- updated_step3_search_state.png")
	log.Println("- updated_step4_final_state.png")
	
	log.Println("\nNote: ITANDI BB requires phone verification for access.")
	log.Println("The scraper can automate the interface navigation but phone verification")
	log.Println("requires manual intervention with actual phone calls.")

	// Keep browser open for manual inspection if not headless
	if !headless {
		log.Println("\nKeeping browser open for 30 seconds for manual inspection...")
		time.Sleep(30 * time.Second)
	}
}